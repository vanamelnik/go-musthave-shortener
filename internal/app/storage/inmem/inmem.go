// Пакет inmem представляет собой реализацию хранилища ключей в виде потокобезопасной in-memory структуры.
// Данные хранилища переиодически сохраняются в файл сервисом gobber.
package inmem

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/storage"
)

var _ storage.Storage = (*DB)(nil)

type (
	row struct {
		SessionID   uuid.UUID
		OriginalURL string
		Key         string
		Deleted     bool
	}

	// DB - реализация интерфейса storage.Storage c thread-safe inmemory хранилищем (структура с RW Mutex).
	DB struct {
		sync.RWMutex

		// repo - in-memory хранилище
		repo []row

		// fileName - имя файла, который хранит данные надиске в формате gob. При старте сервиса in-memory
		// хранилище загружается из файла и по ходу работы периодически переписывает файл, если были изменения.
		fileName string

		// flushInterval - интервал по истечении которого происходит обновление файла, если
		// флаг isChanged установлен.
		flushInterval time.Duration

		// isChanged флаг устанавливается при любых изменениях в хранилище и служит сигналом к тому, что
		// файл с данными необходимо обновить.
		isChanged bool

		gobberStop chan struct{}
	}
)

// New инициализирует структуру in-memory хранилища.
func NewDB(fileName string, interval time.Duration) (*DB, error) {
	if err := validate(fileName, interval); err != nil {
		return nil, err
	}
	repo, err := initRepo(fileName)
	if err != nil {
		return nil, err
	}

	db := &DB{
		repo:          repo,
		fileName:      fileName,
		flushInterval: interval,
		isChanged:     false,
		gobberStop:    make(chan struct{}),
	}

	go db.gobber()

	return db, nil
}

// Close закрывает сервис in-memory хранилища и останавливает воркер gobber.
func (db *DB) Close() {
	db.flush()

	db.Lock()
	defer db.Unlock()

	if db.gobberStop != nil {
		close(db.gobberStop)
	}
	db.gobberStop = nil
}

func validate(filename string, flushinterval time.Duration) error {
	var err *multierror.Error
	if filename == "" {
		err = multierror.Append(err, errors.New("missing file name"))
	}
	if flushinterval == 0 {
		err = multierror.Append(err, errors.New("invalid flush interval"))
	}
	return err.ErrorOrNil()
}

// Store сохраняет в репозитории пару ключ:url.
// если ключ уже используется, выдается ошибка.
func (db *DB) Store(ctx context.Context, id uuid.UUID, key, url string) error {
	if db.hasKey(ctx, key) {
		return fmt.Errorf("DB: the key %s already in use", key)
	}
	if exitingKey, ok := db.hasURL(url); ok {
		return &storage.ErrURLArlreadyExists{
			Key: exitingKey,
			URL: url,
		}
	}

	db.Lock()
	defer db.Unlock()
	db.repo = append(db.repo, row{
		SessionID:   id,
		OriginalURL: url,
		Key:         key,
	})
	db.isChanged = true

	return nil
}

// hasKey проверяет наличие в базе записи с ключом key.
func (db *DB) hasKey(ctx context.Context, key string) bool {
	_, err := db.Get(ctx, key)
	return err == nil
}

// hasUrl проверяет в базе записи с переданным url и в случае успеха возвращает ключ
func (db *DB) hasURL(url string) (key string, ok bool) {
	db.RLock()
	defer db.RUnlock()

	for _, rec := range db.repo {
		if rec.OriginalURL == url && !rec.Deleted {
			return rec.Key, true
		}
	}

	return "", false
}

// Get извлекает из хранилища длинный url по ключу.
// Если ключа в базе нет, возвращается ошибка.
func (db *DB) Get(ctx context.Context, key string) (string, error) {
	db.RLock()
	defer db.RUnlock()

	for _, r := range db.repo {
		if r.Key == key {
			if r.Deleted {
				return "", storage.ErrDeleted
			}
			return r.OriginalURL, nil
		}
	}

	return "", fmt.Errorf("DB: key %s not found", key)
}

// GetAll является реализацией метода GetAll интерфейса storage.Storage
func (db *DB) GetAll(ctx context.Context, id uuid.UUID) map[string]string {
	list := make(map[string]string)

	db.RLock()
	defer db.RUnlock()

	for _, r := range db.repo {
		if r.SessionID == id && !r.Deleted {
			list[r.Key] = r.OriginalURL
		}
	}
	return list
}

// BatchStore - реализация метода интерфейса storage.Storage.
func (db *DB) BatchStore(ctx context.Context, id uuid.UUID, records []storage.Record) error {
	db.Lock()
	defer db.Unlock()

	// В случае обнаружения совпадений отменяем всю транзакцию
	tmpRepo := make([]row, 0, len(records))
	for _, rec := range records {
		for _, r := range db.repo {
			if r.Key == rec.Key {
				return fmt.Errorf("DB: key %s already defined", rec.Key)
			}
			if r.OriginalURL == rec.OriginalURL {
				return storage.ErrBatchURLUniqueViolation
			}
		}
		tmpRepo = append(tmpRepo, row{
			SessionID:   id,
			OriginalURL: rec.OriginalURL,
			Key:         rec.Key,
		})
	}
	// делаем "коммит транзакции"
	db.repo = append(db.repo, tmpRepo...)
	db.isChanged = true

	return nil
}

// BatchDelete - реализация метода интерфейса storage.Storage.
func (db *DB) BatchDelete(ctx context.Context, id uuid.UUID, keys []string) error {
	db.Lock()
	defer db.Unlock()

	for _, key := range keys {
		for i, r := range db.repo {
			if r.Key == key && r.SessionID == id {
				db.repo[i].Deleted = true
				db.isChanged = true
			}
		}
	}

	return nil
}

// Stats - реализация метода интерфейса storage.Storage.
func (db *DB) Stats(ctx context.Context) (urls int, users int, err error) {
	urls = 0
	userMap := make(map[uuid.UUID]struct{})
	for _, row := range db.repo {
		if row.Deleted { // не учитываем удаленные записи
			continue
		}
		urls++
		userMap[row.SessionID] = struct{}{}
	}

	return urls, len(userMap), nil
}

func (db *DB) Ping() error {
	return nil
}
