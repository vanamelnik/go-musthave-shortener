package storage

import "github.com/google/uuid"

// Storage представляет хранилище для  пар key:URL.
type (
	Storage interface {
		// Store сохраняет в хранилище пару ключ:url и возвращает ошибку, если ключ уже используется.
		Store(id uuid.UUID, key, url string) error
		// Get по ключу возвращает значение, либо ошибку, если ключа в базе нет.
		Get(key string) (string, error)
		// GetAll возвращает все пары <key>:<URL> созданные данным пользователем.
		// Если ни одной записи не найдено, возвращается пустая мапа.
		GetAll(id uuid.UUID) map[string]string
		// BatchStore сохраняет в хранилище пакет с парами <OriginalURL> : <Key> из передаваемых объектов Record.
		// Ошибка выдается, если хотя бы один ключ не уникален.
		BatchStore(id uuid.UUID, records []Record) error
		// Close  завершает работу хранилища
		Close()
		// Ping проверяет соединение с хранилищем
		Ping() error
	}

	// Record хранит информацию, передаваемую базе данных в пакетном запросе.
	Record struct {
		// CorellationID - строковый идентификатор. В хранилище не сохраняется.
		CorellationID string
		// OriginalURL - передаваемый URL для сокращения
		OriginalURL string
		// Key - ключ для доступа к оригинальному URL
		Key string
	}

	ErrURLArlreadyExists struct {
		Key string
		Url string
	}
)

func (err ErrURLArlreadyExists) Error() string {
	return "Url already exists in the database"
}
