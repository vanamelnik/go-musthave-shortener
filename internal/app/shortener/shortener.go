package shortener

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

// Длина ключа короткого адреса
const keyLength = 8

// DB представляет хранилище для  пар key:URL
type DB interface {
	Store(key, url string) error    // сохраняет в хранилище пару ключ:url и возвращает ошибку, если ключ уже используется
	Get(key string) (string, error) // по ключу возвращает значение, либо ошибку, если ключа в базе нет.
}

// Shortener - сервис создания, хранения и получения коротких URL адресов
type Shortener struct {
	db DB
}

// NewShortener инициализирует новую структуру Shortener с использованием заданного хранилища
func NewShortener(db DB) *Shortener {
	return &Shortener{
		db: db,
	}
}

// ShortenURL принимает в теле запроса URL, генерирует для него рандомный ключ, производит проверку его уникальности,
// сохраняет его в БД и возвращает строку сокращенного url в теле ответа
//
// POST /
func (s *Shortener) ShortenURL(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Printf("shortener: ShortenURL: %v", err)
		http.Error(w, "Sorry, something went worng...", http.StatusInternalServerError)
		return
	}
	url := string(body)
	if url == "" {
		log.Println("shortener: no url in request")
		http.Error(w, "Wrong url", http.StatusBadRequest)
		return
	}
	// цикл проверки уникальности
	for {
		key := generateKey()
		if _, err := s.db.Get(key); err != nil {
			s.db.Store(key, url) // не проверяем ошибку, т.к. уникальность ключа только что проверена
			log.Printf("shortener: ShortenURL: created a token %v for %v", key, url)
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, "http://localhost:8080/%s", key)
			return
		}
		log.Printf("Wow!!! %d-значный случайный код повторился! Совпадение? Не думаю!", keyLength)
	}

}

// DecodeURL принимает короткий параметр и производит редирект на изначальный url с кодом 307
//
// GET /{id}
func (s *Shortener) DecodeURL(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/")
	if len(key) != 8 {
		log.Printf("shortener: DecodeURL: wrong key %v", key)
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
	// ***вопрос*** если на входе в сервис подавался URL без префикса 'http[s]://', (например 'yandex.ru') то редирект происходит по неверному
	// адресу http://localhost:8080/yandex.ru
	// есть ли способ обойти это, кроме как валидация входящих URL?
	w.Header().Add("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
	// или http.Redirect(w, r, url, http.StatusTemporaryRedirect) - есть ли разница?
}

// generateKey создает рандомную строку из строчных букв и цифр. Длина строки задана в глобальной переменной keyLength
func generateKey() string {
	const chars = "abcdefghijklmnopqrstuvwxyz1234567890"
	buf := make([]byte, keyLength)
	for i := range buf {
		buf[i] = chars[rand.Intn(len(chars))]
	}
	return string(buf)
}
