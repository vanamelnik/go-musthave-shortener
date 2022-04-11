package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Storage представляет хранилище для  пар key:URL.
type (
	Storage interface {
		// Store сохраняет в хранилище пару ключ:url и возвращает ошибку, если ключ уже используется.
		Store(ctx context.Context, id uuid.UUID, key, url string) error
		// Get по ключу возвращает значение, либо ошибку, если ключа в базе нет.
		Get(ctx context.Context, key string) (string, error)
		// GetAll возвращает все пары <key>:<URL> созданные данным пользователем.
		// Если ни одной записи не найдено, возвращается пустая мапа.
		GetAll(ctx context.Context, id uuid.UUID) map[string]string
		// BatchStore сохраняет в хранилище пакет с парами <OriginalURL> : <Key> из передаваемых объектов Record.
		// Ошибка выдается, если хотя бы один ключ не уникален.
		BatchStore(ctx context.Context, id uuid.UUID, records []Record) error
		// BatchDelete производит мягкое удаление записей из хранилища с ключами <keys>, если их создал пользователь
		// с указанным id.
		BatchDelete(ctx context.Context, id uuid.UUID, keys []string) error
		// Stats возвращает общее количество сокращенных URL и количество пользователей в сервисе.
		Stats(ctx context.Context) (urls int, users int, err error)
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

	storageError string

	// ErrURLArlreadyExists возвращается при попытке сохранить в базу URL, который в ней уже сохранён.
	ErrURLArlreadyExists struct {
		Key string
		URL string
	}
)

func (e storageError) Error() string {
	return string(e)
}

func (err ErrURLArlreadyExists) Error() string {
	return fmt.Sprintf("Url %s already exists in the database", err.URL)
}

const (
	// ErrBatchURLUniqueViolation возвращается при попытке пакетного сохранения URL, если некоторые из них уже есть в базе.
	ErrBatchURLUniqueViolation storageError = "Some of URLs is already exists in the database"

	// ErrDeleted возвращается, когда запрашиваемый ключ был удален.
	ErrDeleted storageError = "Key was deleted"
)
