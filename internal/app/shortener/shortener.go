package shortener

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/context"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/storage"

	"github.com/gorilla/mux"
)

// keyLength определяет длину ключа короткого адреса.
const keyLength = 8

// Shortener - сервис создания, хранения и получения коротких URL адресов.
type Shortener struct {
	db      storage.Storage
	BaseURL string
}

// NewShortener инициализирует новую структуру Shortener с использованием заданного хранилища.
func NewShortener(baseURL string, db storage.Storage) *Shortener {
	return &Shortener{
		BaseURL: baseURL,
		db:      db,
	}
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
	shortURL, err := s.shortenURL(w, id, urlReq.URL)
	if err != nil {
		log.Printf("APIShortenURL: %v", err)
		http.Error(w, "Wrong URL", http.StatusBadRequest)

		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
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
	shortURL, err := s.shortenURL(w, id, string(body))
	if err != nil {
		http.Error(w, "Wrong URL", http.StatusBadRequest)
		log.Printf("shortener: %v", err)

		return
	}
	w.WriteHeader(http.StatusCreated)
	// nolint:errcheck
	w.Write([]byte(shortURL))
}
func (s Shortener) shortenURL(w http.ResponseWriter, id uuid.UUID, u string) (shortURL string, retErr error) {
	url, err := checkURL(u)
	if err != nil {

		return "", err
	}
	// цикл проверки уникальности
	for {
		key := generateKey()
		if _, err := s.db.Get(key); err != nil {
			err = s.db.Store(id, key, url.String())
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
	url, err := s.db.Get(key)
	if err != nil {
		log.Printf("shortener: DecodeURL: could not find url with key %v", key)
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

	list := s.db.GetAll(id)
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
