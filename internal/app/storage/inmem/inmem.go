package inmem

import (
	"fmt"
	"sync"

	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/storage"
)

var _ storage.Storage = (*DB)(nil)

// DB - реализация интерфейса DB c thread-safe inmemory хранилищем (map с RW Mutex).
type DB struct {
	sync.RWMutex

	repo map[string]string // [key]url
}

// New инициализирует структуру in-memory хранилища.
func NewDB() *DB {
	return &DB{
		repo: make(map[string]string),
	}
}

// Store сохраняет в репозитории пару ключ:url.
// если ключ уже используется, выдается ошибка.
func (d *DB) Store(key, url string) error {
	if d.Has(key) {
		return fmt.Errorf("DB: the key %s already in use", key)
	}
	d.Lock()
	defer d.Unlock()
	d.repo[key] = url

	return nil
}

// Has проверяет наличие в базе записи с ключом key.
func (d *DB) Has(key string) bool {
	d.RLock()
	defer d.RUnlock()
	_, ok := d.repo[key]

	return ok
}

// Get извлекает из хранилища длинный url по ключу.
// Если ключа в базе нет, возвращается ошибка.
func (d *DB) Get(key string) (string, error) {
	d.RLock()
	defer d.RUnlock()
	url, ok := d.repo[key]
	if !ok {
		return "", fmt.Errorf("DB: key %s not found", key)
	}

	return url, nil
}
