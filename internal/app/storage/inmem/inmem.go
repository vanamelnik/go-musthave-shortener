package inmem

import (
	"fmt"
	"sync"
	"time"

	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/storage"
)

var _ storage.Storage = (*DB)(nil)

// DB - реализация интерфейса DB c thread-safe inmemory хранилищем (map с RW Mutex).
type DB struct {
	sync.RWMutex

	// repo - in-memory хранилище пар ключ-URL
	repo map[string]string // [key]url

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
func (db *DB) Store(key, url string) error {
	if db.Has(key) {
		return fmt.Errorf("DB: the key %s already in use", key)
	}
	db.Lock()
	defer db.Unlock()
	db.repo[key] = url
	db.isChanged = true

	return nil
}

// Has проверяет наличие в базе записи с ключом key.
func (db *DB) Has(key string) bool {
	db.RLock()
	defer db.RUnlock()
	_, ok := db.repo[key]

	return ok
}

// Get извлекает из хранилища длинный url по ключу.
// Если ключа в базе нет, возвращается ошибка.
func (db *DB) Get(key string) (string, error) {
	db.RLock()
	defer db.RUnlock()
	url, ok := db.repo[key]
	if !ok {
		return "", fmt.Errorf("DB: key %s not found", key)
	}

	return url, nil
}
