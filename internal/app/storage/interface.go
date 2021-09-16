package storage

import "github.com/google/uuid"

// Storage представляет хранилище для  пар key:URL.
type Storage interface {
	// Store сохраняет в хранилище пару ключ:url и возвращает ошибку, если ключ уже используется.
	Store(id uuid.UUID, key, url string) error
	// Get по ключу возвращает значение, либо ошибку, если ключа в базе нет.
	Get(key string) (string, error)
	// GetAll возвращает все пары <key>:<URL> созданные данным пользователем.
	// Если ни одной записи не найдено, возвращается пустая мапа.
	GetAll(id uuid.UUID) map[string]string
}
