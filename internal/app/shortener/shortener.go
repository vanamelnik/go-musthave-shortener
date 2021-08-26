package shortener

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"

	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/storage"

	"github.com/gorilla/mux"
)

// keyLength определяет длину ключа короткого адреса.
const keyLength = 8

// Shortener - сервис создания, хранения и получения коротких URL адресов.
type Shortener struct {
	db   storage.Storage
	port string
	host string
}

// NewShortener инициализирует новую структуру Shortener с использованием заданного хранилища
func NewShortener(host, port string, db storage.Storage) *Shortener {
	return &Shortener{
		port: port,
		host: host,
		db:   db,
	}
}

// ShortenURL принимает в теле запроса URL, генерирует для него рандомный ключ, производит проверку его уникальности,
// сохраняет его в БД и возвращает строку сокращенного url в теле ответа.
//
// POST /
func (s *Shortener) ShortenURL(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("shortener: ShortenURL: %v", err)
		http.Error(w, "Sorry, something went worng...", http.StatusInternalServerError)

		return
	}
	url, err := url.Parse(string(body))
	if url.Host == "" || err != nil {
		log.Println("shortener: no url in request")
		http.Error(w, "Wrong url", http.StatusBadRequest)

		return
	}
	// цикл проверки уникальности
	for {
		key := generateKey()
		if _, err := s.db.Get(key); err != nil {
			// nolint:errcheck
			s.db.Store(key, url.String()) // не проверяем ошибку, т.к. уникальность ключа только что проверена.
			log.Printf("shortener: ShortenURL: created a token %v for %v", key, url)
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, "%s%s/%s", s.host, s.port, key)
			return
		}
		log.Printf("Wow!!! %d-значный случайный код повторился! Совпадение? Не думаю!", keyLength)
	}

}

// DecodeURL принимает короткий параметр и производит редирект на изначальный url с кодом 307.
//
// GET /{id}
func (s *Shortener) DecodeURL(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Add("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
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
