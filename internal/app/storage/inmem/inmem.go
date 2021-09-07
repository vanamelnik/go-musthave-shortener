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

	gobberStop chan bool
}

// New инициализирует структуру in-memory хранилища.
func NewDB(fileName string, interval time.Duration) *DB {
	repo, err := readRepo(fileName)
	if err != nil {
		repo = make(map[string]string)
	}

	db := &DB{
		repo:          repo,
		fileName:      fileName,
		flushInterval: interval,
		isChanged:     false,
		gobberStop:    make(chan bool),
	}

	go db.gobber()

	return db
}

func (db *DB) Close() {
	db.gobberStop <- true
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
	d.isChanged = true

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
