package app

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
)

// Длина ключа короткого адреса
const keyLength = 8

// Shortener представляет thread-safe сервис сохранения пар {key : URL} в map
type Shortener struct {
	sync.RWMutex

	database map[string]string // [key]URL
}

func NewShortener() *Shortener {
	return &Shortener{
		database: make(map[string]string),
	}
}

// ShortenURL принимает в теле запроса URL, генерирует для него рандомный ключ, сохраняет его в БД
// и возвращает строку сокращенного url в теле ответа
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
	key := s.createKey()
	s.Lock()
	s.database[key] = url
	s.Unlock()
	log.Printf("shortener: ShortenURL: created a token %v for %v", key, url)

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "http://localhost:8080/%s", key)
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
	s.RLock()
	url, ok := s.database[key]
	s.RUnlock()
	if !ok {
		log.Printf("shortener: DecodeURL: could not find url with key %v", key)
		http.Error(w, "Wrong url", http.StatusBadRequest)
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

// createKey создает рандомную строку из строчных букв и цифр. Длина строки задана в глобальной переменной keyLength
// производится проверка уникальности произведенной строки
func (s *Shortener) createKey() string {
	const chars = "abcdefghijklmnopqrstuvwxyz1234567890"
	buf := make([]byte, keyLength)
	done := false
	// проверка уникальности сгенерированного ключа
	for !done {
		for i := range buf {
			buf[i] = chars[rand.Intn(len(chars))]
		}
		s.RLock()
		if _, ok := s.database[string(buf)]; !ok { // совпадение? не думаю!
			done = true
		} else {
			log.Println("WOW!!! WE DID IT!************** RANDOM KEY REPEATED!!!")
		}
		s.RUnlock()
	}
	return string(buf)
}
