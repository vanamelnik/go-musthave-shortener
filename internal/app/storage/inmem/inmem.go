package inmem

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/storage"
)

var _ storage.Storage = (*DB)(nil)

type row struct {
	SessionID   uuid.UUID
	OriginalURL string
	Key         string
}

// DB - реализация интерфейса storage.Storage c thread-safe inmemory хранилищем (структура с RW Mutex).
type DB struct {
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

// New инициализирует структуру in-memory хранилища.
func NewDB(fileName string, interval time.Duration) (*DB, error) {
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

func (db *DB) Close() {
	db.flush()

	db.Lock()
	defer db.Unlock()

	if db.gobberStop != nil {
		close(db.gobberStop)
	}
	db.gobberStop = nil
}

// Store сохраняет в репозитории пару ключ:url.
// если ключ уже используется, выдается ошибка.
func (db *DB) Store(id uuid.UUID, key, url string) error {
	if db.hasKey(key) {
		return fmt.Errorf("DB: the key %s already in use", key)
	}
	if exitingKey, ok := db.hasURL(url); ok {
		return &storage.ErrURLArlreadyExists{
			Key: exitingKey,
			Url: url,
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
func (db *DB) hasKey(key string) bool {
	_, err := db.Get(key)
	return err == nil
}

// hasUrl проверяет в базе записи с переданным url и в случае успеха возвращает ключ
func (db *DB) hasURL(url string) (key string, ok bool) {
	db.RLock()
	defer db.RUnlock()

	for _, rec := range db.repo {
		if rec.OriginalURL == url {
			return rec.Key, true
		}
	}

	return "", false
}

// Get извлекает из хранилища длинный url по ключу.
// Если ключа в базе нет, возвращается ошибка.
func (db *DB) Get(key string) (string, error) {
	db.RLock()
	defer db.RUnlock()

	for _, r := range db.repo {
		if r.Key == key {
			return r.OriginalURL, nil
		}
	}

	return "", fmt.Errorf("DB: key %s not found", key)
}

// GetAll является реализацией метода GetAll интерфейса storage.Storage
func (db *DB) GetAll(id uuid.UUID) map[string]string {
	list := make(map[string]string)

	db.RLock()
	defer db.RUnlock()

	for _, r := range db.repo {
		if r.SessionID == id {
			list[r.Key] = r.OriginalURL
		}
	}
	return list
}

// BatchStore - реализация метода интерфейса storage.Storage
func (db *DB) BatchStore(id uuid.UUID, records []storage.Record) error {
	db.Lock()
	defer db.Unlock()

	// не используем методы Has и Store, чтобы много раз не лочить / разлочивать... Или ну его?
	for _, rec := range records {
		for _, r := range db.repo {
			if r.Key == rec.Key {
				return fmt.Errorf("DB: key %s already defined", rec.Key)
			}
		}
		db.repo = append(db.repo, row{
			SessionID:   id,
			OriginalURL: rec.OriginalURL,
			Key:         rec.Key,
		})
		db.isChanged = true
	}

	return nil
}

func (db *DB) Ping() error {
	return nil
}
