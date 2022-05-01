package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/context"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/shortener"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/storage"
)

type Rest struct {
	shortener *shortener.Shortener
}

func NewRest(s *shortener.Shortener) Rest {
	return Rest{shortener: s}
}

// Ping проверяет соединение с базой данных.
//
// GET /ping
func (rest Rest) Ping(w http.ResponseWriter, r *http.Request) {
	if err := rest.shortener.Ping(); err != nil {
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
func (rest Rest) APIShortenURL(w http.ResponseWriter, r *http.Request) {
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
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&urlReq); err != nil {
		log.Printf("APIShortenURL: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}
	shortURL, err := rest.shortener.ShortenURL(r.Context(), id, urlReq.URL)
	statusCode := http.StatusCreated
	if err != nil {
		log.Printf("APIShortenURL: %v", err)
		var errURLAlreadyExists *storage.ErrURLArlreadyExists

		if errors.As(err, &errURLAlreadyExists) {
			statusCode = http.StatusConflict
			shortURL = fmt.Sprintf("%s/%s", rest.shortener.BaseURL, errURLAlreadyExists.Key)
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
func (rest Rest) ShortenURL(w http.ResponseWriter, r *http.Request) {
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
	shortURL, err := rest.shortener.ShortenURL(r.Context(), id, string(body))
	if err != nil {
		log.Printf("shortener: %v", err)
		var errURLAlreadyExists *storage.ErrURLArlreadyExists
		if errors.As(err, &errURLAlreadyExists) {
			w.WriteHeader(http.StatusConflict)
			shortURL = fmt.Sprintf("%s/%s", rest.shortener.BaseURL, errURLAlreadyExists.Key)
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

// DecodeURL принимает короткий параметр и производит редирект на изначальный url с кодом 307.
//
// GET /{id}
func (rest Rest) DecodeURL(w http.ResponseWriter, r *http.Request) {
	key, ok := mux.Vars(r)["id"]
	if !ok || len(key) != 8 {
		log.Printf("shortener: DecodeURL: wrong key '%v'", key)
		http.Error(w, "Wrong key", http.StatusBadRequest)

		return
	}
	url, err := rest.shortener.DecodeURL(r.Context(), key)
	if err != nil {
		log.Printf("shortener: DecodeURL: could not find url with key %v: %v", key, err)
		if errors.Is(err, storage.ErrDeleted) {
			http.Error(w, "URL was deleted", http.StatusGone)
			return
		}

		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}
	// log.Printf("shortener: DecodeURL: redirecting to %v (key: %v)", url, key)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// UserURLs возвращает в ответе json с массивом записей всех URL, созданных текущем пользователем
//
// GET /api/user/urls
func (rest Rest) UserURLs(w http.ResponseWriter, r *http.Request) {
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

	list := rest.shortener.GetAll(r.Context(), id)
	log.Printf("[INF] shortener: requested entries for id=%s, found %d items.", id, len(list))
	if len(list) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	userURLs := make([]urlRec, 0, len(list))
	for key, url := range list {
		userURLs = append(userURLs, urlRec{
			ShortURL:    fmt.Sprintf("%s/%s", rest.shortener.BaseURL, key),
			OriginalURL: url,
		})
	}

	w.Header().Add("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	if err := enc.Encode(userURLs); err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		log.Printf("shortener: %v", err)

		return
	}
}

// BatchShortenURL формирует ключи для переданных через тело запроса URL и передает данные на сохранение в базу данных.
//
// POST /api/shorten/batch
func (rest Rest) BatchShortenURL(w http.ResponseWriter, r *http.Request) {
	id, err := context.ID(r.Context()) // Значение uuid добавлено в контекст запроса middleware'й.
	if err != nil {
		log.Printf("shortener: Batch: %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)

		return
	}
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	batchReq := make([]shortener.BatchShortenRequest, 0)
	if err = dec.Decode(&batchReq); err != nil {
		log.Printf("shortener: Batch: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	resp, err := rest.shortener.BatchShortenURL(r.Context(), id, batchReq)
	if err != nil {
		if errors.Is(err, storage.ErrBatchURLUniqueViolation) {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
		}
		log.Printf("shortener: Batch: cannot store the records: %v", err)

		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		log.Printf("shortener: Batch: %v", err)

		return
	}
}

// DeleteURLs удаляет все записи о ключах, созданных в рамках текущей сессии.
// Ключи передаются в формате ["<key1>", "<key2>"...]
//
// DELETE /api/user/urls
func (rest Rest) DeleteURLs(w http.ResponseWriter, r *http.Request) {
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

	if err := rest.shortener.BatchDelete(r.Context(), id, keys); err != nil {
		log.Printf("shortener: delete: %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// Stats предоставляет информацию о количестве сокращенных URL и о количестве пользователей.
// информация предоставляется только по запросу с доверенной подсети.
//
// GET /api/internal/stats
func (rest Rest) Stats(w http.ResponseWriter, r *http.Request) {
	type stats struct {
		URLs  int `json:"urls"`
		Users int `json:"users"`
	}
	urls, users, err := rest.shortener.Stats(r.Context())
	if err != nil {
		log.Printf("shortener: stats: %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)

		return
	}
	st := stats{
		URLs:  urls,
		Users: users,
	}
	if err := json.NewEncoder(w).Encode(st); err != nil {
		log.Printf("shortener: stats: %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)

		return
	}
}
