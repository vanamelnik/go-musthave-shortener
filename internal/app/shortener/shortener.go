// В пакете shortener представлены обработчики вызовов REST API.
package shortener

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/url"

	"github.com/google/uuid"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/dataloader"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/storage"
)

// keyLength определяет длину ключа короткого адреса.
const keyLength = 8

// Shortener - сервис создания, хранения и получения коротких URL адресов.
type (
	Shortener struct {
		db      storage.Storage
		BaseURL string

		dl dataloader.DataLoader
	}

	BatchShortenRequest struct {
		CorrelationID string `json:"correlation_id"`
		OriginalURL   string `json:"original_url"`
	}
	BatchShortenResponse struct {
		CorrelationID string `json:"correlation_id"`
		ShortURL      string `json:"short_url"`
	}
)

// NewShortener инициализирует новую структуру Shortener с использованием заданного хранилища.
func NewShortener(baseURL string, db storage.Storage, dl dataloader.DataLoader) *Shortener {
	return &Shortener{
		BaseURL: baseURL,
		db:      db,
		dl:      dl,
	}
}

// Ping проверяет соединение с базой данных.
func (s Shortener) Ping() error {
	return s.db.Ping()
}

// ShortenURL генерирует для переданного URL рандомный ключ, производит проверку его уникальности
// и сохраняет в хранилище.
func (s Shortener) ShortenURL(ctx context.Context, id uuid.UUID, urlStr string) (shortURL string, retErr error) {
	url, err := checkURL(urlStr)
	if err != nil {
		return "", err
	}

	// цикл проверки уникальности
	for {
		key := generateKey()
		if _, err := s.db.Get(ctx, key); err != nil {
			err = s.db.Store(ctx, id, key, url.String())
			if err != nil {
				return "", err
			}
			// log.Printf("[INF] shortener: ShortenURL: created a token %v for %v", key, url)
			shortURL = fmt.Sprintf("%s/%s", s.BaseURL, key)

			return shortURL, nil
		}
		log.Printf("Wow!!! %d-значный случайный код повторился! Совпадение? Не думаю!", keyLength)
	}
}

// DecodeURL возвращает изначальный URL по ключу.
func (s Shortener) DecodeURL(ctx context.Context, key string) (string, error) {
	return s.db.Get(ctx, key)
}

// GetAll возвращает записи всех URL, созданных пользователем с переданным id.
func (s Shortener) GetAll(ctx context.Context, id uuid.UUID) map[string]string {
	return s.db.GetAll(ctx, id)
}

// BatchShortenURL формирует ключи для переданных чURL и передает данные на сохранение в базу данных.
func (s Shortener) BatchShortenURL(ctx context.Context, id uuid.UUID, request []BatchShortenRequest) ([]BatchShortenResponse, error) {
	records := make([]storage.Record, 0, len(request))
	for _, rec := range request {
		records = append(records, storage.Record{
			CorellationID: rec.CorrelationID,
			OriginalURL:   rec.OriginalURL,
			Key:           generateKey(),
		})
	}

	if err := s.db.BatchStore(ctx, id, records); err != nil {
		return nil, err
	}

	batchResp := make([]BatchShortenResponse, len(records))
	for i, rec := range records {
		batchResp[i] = BatchShortenResponse{
			CorrelationID: rec.CorellationID,
			ShortURL:      fmt.Sprintf("%s/%s", s.BaseURL, rec.Key),
		}
	}
	log.Printf("shortener: Batch: successfully added %d records to the repository", len(records))

	return batchResp, nil
}

// BatchDelete удаляет указанные записи о ключах, созданных пользователем с переданным id.
func (s Shortener) BatchDelete(ctx context.Context, id uuid.UUID, keys []string) error {
	return s.dl.BatchDelete(ctx, id, keys)
}

// Stats предоставляет информацию о количестве сокращенных URL и о количестве пользователей.
func (s Shortener) Stats(ctx context.Context) (urls int, users int, err error) {
	return s.db.Stats(ctx)
}

// generateKey создает рандомную строку из строчных букв и цифр. Длина строки задана в глобальной переменной keyLength.
func generateKey() string {
	const chars = "abcdefghijklmnopqrstuvwxyz1234567890"
	buf := make([]byte, keyLength)
	for i := range buf {
		buf[i] = chars[rand.Intn(len(chars))]
	}

	return string(buf)
}

// checkURL проверяет входящую строку, является ли она URL с полями scheme и host.
func checkURL(u string) (*url.URL, error) {
	url, err := url.Parse(u)
	if err != nil {

		return nil, err
	}
	if url.Host == "" || url.Scheme == "" {

		return nil, fmt.Errorf("wrong URL: %s", u)
	}

	return url, nil
}
