package storage

// Storage представляет хранилище для  пар key:URL.
type Storage interface {
	// Store сохраняет в хранилище пару ключ:url и возвращает ошибку, если ключ уже используется.
	Store(key, url string) error
	// Get по ключу возвращает значение, либо ошибку, если ключа в базе нет.
	Get(key string) (string, error)
}
