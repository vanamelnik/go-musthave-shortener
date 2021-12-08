package shortener

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/context"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/dataloader"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/storage"

	"github.com/gorilla/mux"
)

// keyLength определяет длину ключа короткого адреса.
const keyLength = 8

// Shortener - сервис создания, хранения и получения коротких URL адресов.
type Shortener struct {
	db      storage.Storage
	BaseURL string

	dl dataloader.DataLoader
}

// NewShortener инициализирует новую структуру Shortener с использованием заданного хранилища.
func NewShortener(baseURL string, db storage.Storage, dl dataloader.DataLoader) *Shortener {
	return &Shortener{
		BaseURL: baseURL,
		db:      db,
		dl:      dl,
	}
}

// Ping проверяет соединение с базой данных.
//
// GET /ping
func (s Shortener) Ping(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(); err != nil {
		log.Printf("storage: ping: %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)

		return
	}
	log.Println("storage: ping OK")
}

// APIShortenURL принимает в теле запроса JSON-объект в формате {"url": "<some_url>"} и
// возвращает в ответе объект {"result": "<shorten_url>"}
//
// POST /api/shorten
func (s Shortener) APIShortenURL(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		URL string `json:"url"`
	}
	type Result struct {
		Result string `json:"result"`
	}
	id, err := context.ID(r.Context()) // Значение uuid добавлено в контекст запроса middleware'й.
	if err != nil {
		log.Printf("shortener: %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)

		return
	}

	urlReq := Request{}
	dec := json.NewDecoder(r.Body)
	defer r.Body.Close()
	if err := dec.Decode(&urlReq); err != nil {
		log.Printf("APIShortenURL: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}
	shortURL, err := s.shortenURL(w, r, id, urlReq.URL)
	statusCode := http.StatusCreated
	if err != nil {
		log.Printf("APIShortenURL: %v", err)
		var errURLAlreadyExists *storage.ErrURLArlreadyExists

		if errors.As(err, &errURLAlreadyExists) {
			statusCode = http.StatusConflict
			shortURL = fmt.Sprintf("%s/%s", s.BaseURL, errURLAlreadyExists.Key)
		} else {
			http.Error(w, "Wrong URL", http.StatusBadRequest)

			return
		}
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	enc := json.NewEncoder(w)
	err = enc.Encode(&Result{Result: shortURL})
	if err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		log.Printf("APIShortHandler: %v", err)
	}
}

// ShortenURL принимает в теле запроса URL, генерирует для него рандомный ключ, производит проверку его уникальности,
// сохраняет его в БД и возвращает строку сокращенного url в теле ответа.
//
// POST /
func (s Shortener) ShortenURL(w http.ResponseWriter, r *http.Request) {
	id, err := context.ID(r.Context()) // Значение uuid добавлено в контекст запроса middleware'й.
	if err != nil {
		log.Printf("shortener: %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)

		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("shortener: ShortenURL: %v", err)
		http.Error(w, "Sorry, something went wrong...", http.StatusInternalServerError)

		return
	}
	shortURL, err := s.shortenURL(w, r, id, string(body))
	if err != nil {
		log.Printf("shortener: %v", err)
		var errURLAlreadyExists *storage.ErrURLArlreadyExists
		if errors.As(err, &errURLAlreadyExists) {
			w.WriteHeader(http.StatusConflict)
			shortURL = fmt.Sprintf("%s/%s", s.BaseURL, errURLAlreadyExists.Key)
			// nolint:errcheck
			w.Write([]byte(shortURL))

			return
		}
		http.Error(w, "Wrong URL", http.StatusBadRequest)

		return
	}
	w.WriteHeader(http.StatusCreated)
	// nolint:errcheck
	w.Write([]byte(shortURL))
}
func (s Shortener) shortenURL(w http.ResponseWriter, r *http.Request, id uuid.UUID, u string) (shortURL string, retErr error) {
	url, err := checkURL(u)
	if err != nil {
		return "", err
	}

	// цикл проверки уникальности
	ctx := r.Context()
	for {
		key := generateKey()
		if _, err := s.db.Get(ctx, key); err != nil {
			err = s.db.Store(ctx, id, key, url.String())
			if err != nil {
				return "", err
			}
			log.Printf("[INF] shortener: ShortenURL: created a token %v for %v", key, url)
			shortURL = fmt.Sprintf("%s/%s", s.BaseURL, key)

			return shortURL, nil
		}
		log.Printf("Wow!!! %d-значный случайный код повторился! Совпадение? Не думаю!", keyLength)
	}
}

// DecodeURL принимает короткий параметр и производит редирект на изначальный url с кодом 307.
//
// GET /{id}
func (s Shortener) DecodeURL(w http.ResponseWriter, r *http.Request) {
	key, ok := mux.Vars(r)["id"]
	if !ok || len(key) != 8 {
		log.Printf("shortener: DecodeURL: wrong key '%v'", key)
		http.Error(w, "Wrong key", http.StatusBadRequest)

		return
	}
	url, err := s.db.Get(r.Context(), key)
	if err != nil {
		log.Printf("shortener: DecodeURL: could not find url with key %v: %v", key, err)
		if errors.Is(err, storage.ErrDeleted) {
			http.Error(w, "URL was deleted", http.StatusGone)
			return
		}

		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}
	log.Printf("shortener: DecodeURL: redirecting to %v (key: %v)", url, key)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// UserURLs возвращает в ответе json с массивом записей всех URL, созданных текущем пользователем
//
// GET /user/urls
func (s Shortener) UserURLs(w http.ResponseWriter, r *http.Request) {
	type urlRec struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}

	id, err := context.ID(r.Context()) // Значение uuid добавлено в контекст запроса middleware'й.
	if err != nil {
		log.Printf("shortener: %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)

		return
	}

	list := s.db.GetAll(r.Context(), id)
	log.Printf("[INF] shortener: requested entries for id=%s, found %d items.", id, len(list))
	if len(list) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	userURLs := make([]urlRec, 0, len(list))
	for key, url := range list {
		userURLs = append(userURLs, urlRec{
			ShortURL:    fmt.Sprintf("%s/%s", s.BaseURL, key),
			OriginalURL: url,
		})
	}

	w.Header().Add("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	if err := enc.Encode(userURLs); err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		log.Printf("shortener: %v", err)
	}
}

// BatchShortenURL формирует ключи для переданных через тело запроса URL и передает данные на сохранение в базу данных.
//
// POST /api/shorten/batch
func (s Shortener) BatchShortenURL(w http.ResponseWriter, r *http.Request) {
	type (
		req struct {
			CorrelationID string `json:"correlation_id"`
			OriginalURL   string `json:"original_url"`
		}
		resp struct {
			CorrelationID string `json:"correlation_id"`
			ShortURL      string `json:"short_url"`
		}
	)
	id, err := context.ID(r.Context()) // Значение uuid добавлено в контекст запроса middleware'й.
	if err != nil {
		log.Printf("shortener: Batch: %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)

		return
	}
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	batchReq := make([]req, 0)
	if err = dec.Decode(&batchReq); err != nil {
		log.Printf("shortener: Batch: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	records := make([]storage.Record, 0, len(batchReq))
	for _, rec := range batchReq {
		records = append(records, storage.Record{
			CorellationID: rec.CorrelationID,
			OriginalURL:   rec.OriginalURL,
			Key:           generateKey(),
		})
	}

	if err = s.db.BatchStore(r.Context(), id, records); err != nil {
		if errors.Is(err, storage.ErrBatchURLUniqueViolation) {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
		}
		log.Printf("shortener: Batch: cannot store the records: %v", err)

		return
	}

	batchResp := make([]resp, len(records))
	for i, rec := range records {
		batchResp[i] = resp{
			CorrelationID: rec.CorellationID,
			ShortURL:      fmt.Sprintf("%s/%s", s.BaseURL, rec.Key),
		}
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(w)
	err = enc.Encode(&batchResp)
	if err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		log.Printf("shortener: Batch: %v", err)

		return
	}

	log.Printf("shortener: Batch: successfully added %d records to the repository", len(records))
}

// DeleteURLs
//
// DELETE /api/user/urls
func (s Shortener) DeleteURLs(w http.ResponseWriter, r *http.Request) {
	id, err := context.ID(r.Context()) // Значение uuid добавлено в контекст запроса middleware'й.
	if err != nil {
		log.Printf("shortener: delete: %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)

		return
	}
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("shortener: delete: %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)

		return
	}

	var keys []string
	if err := json.Unmarshal(b, &keys); err != nil {
		log.Printf("shortener: delete: %v", err)
		http.Error(w, "Wrong format", http.StatusBadRequest)

		return
	}

	if err := s.dl.BatchDelete(r.Context(), id, keys); err != nil {
		log.Printf("shortener: delete: %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusAccepted)
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

// checkURL проверяет входящую строку, является ли она URL с полями scheme и host
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
