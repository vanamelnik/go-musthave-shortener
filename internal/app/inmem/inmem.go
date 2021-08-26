package inmem

import (
	"fmt"
	"sync"

	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/shortener"
)

var _ shortener.DB = &DB{}

// DB - реализация интерфейса DB c thread-safe inmemory хранилищем (map с RW Mutex)
type DB struct {
	sync.RWMutex

	repo map[string]string // [key]url
}

// New инициализирует структуру in-memory хранилища
func NewDB() shortener.DB {
	return &DB{
		repo: make(map[string]string),
	}
}

// Store сохраняет в репозитории пару ключ:url
// если ключ уже используется, выдается ошибка
func (d *DB) Store(key, url string) error {
	d.RLock()
	_, ok := d.repo[key]
	d.RUnlock()
	if ok {
		return fmt.Errorf("DB: the key %s already in use", key)
	}
	d.Lock()
	d.repo[key] = url
	d.Unlock()
	return nil
}

// Get извлекает из хранилища длинный url по ключу
// Если ключа в базе нет, возвращается ошибка
func (d *DB) Get(key string) (string, error) {
	d.RLock()
	url, ok := d.repo[key]
	d.RUnlock()
	if !ok {
		return "", fmt.Errorf("DB: key %s not found", key)
	}
	return url, nil
}
