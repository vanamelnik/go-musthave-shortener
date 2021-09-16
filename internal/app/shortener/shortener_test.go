package shortener_test

import (
	"errors"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/context"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/shortener"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/storage"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/storage/inmem"
)

// TestShortener - комплексный тест, прогоняющий все виды запросов к inmemory хранилищу.
func TestShortener(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	type want struct {
		statusCode int
		body       string
		location   string
	}

	// Аргументы для GET тестов (ключ, по которому должен выдаться сохраненный URL).
	// Первые два поля заполняются POST тестами.
	args := []string{
		"",            // заполняется POST тестом №1
		"",            // заполняется POST тестом №2
		"qwertyui",    // этого ключа нет в базе
		"favicon.ico", // ключ неверной длины
	}
	testsPost := []struct {
		name    string
		body    string
		want    want
		saveArg *string //указатель на хранилище для полученного значения
	}{
		{
			name: "Test POST#1  - yandex.ru",
			body: "http://yandex.ru",
			want: want{
				statusCode: http.StatusCreated,
				body:       "",
			},
			saveArg: &args[0],
		},
		{
			name: "Test POST#2 - google.com",
			body: "http://google.com",
			want: want{
				statusCode: http.StatusCreated,
				body:       "",
			},
			saveArg: &args[1],
		},
		{
			name: "Test POST#3 - no body",
			body: "",
			want: want{
				statusCode: http.StatusBadRequest,
				body:       "Wrong URL",
			},
			saveArg: nil,
		},
	}
	testsGet := []struct {
		name string
		arg  *string
		want want
	}{
		{
			name: "Test GET#1 - yandex.ru",
			arg:  &args[0],
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				location:   "http://yandex.ru",
			},
		},
		{
			name: "Test GET#2 - google.com",
			arg:  &args[1],
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				location:   "http://google.com",
			},
		},
		{
			name: "Test GET#3 - no arg",
			arg:  nil,
			want: want{
				statusCode: http.StatusBadRequest,
				body:       "Wrong key",
			},
		},
		{
			name: "Test GET#4 - unknown key (not stored in repo)",
			arg:  &args[2],
			want: want{
				statusCode: http.StatusNotFound,
				body:       "URL not found",
			},
		},
		{
			name: "Test GET#5 - not a key",
			arg:  &args[3],
			want: want{
				statusCode: http.StatusBadRequest,
				body:       "Wrong key",
			},
		},
	}

	db, err := inmem.NewDB("tmp.db", time.Hour)
	require.NoError(t, err)
	defer func() {
		db.Close()
		require.NoError(t, os.Remove("tmp.db"))
	}()

	s := shortener.NewShortener("http://localhost:8080", db)

	// запускаем тесты POST
	for _, tc := range testsPost {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			ctx := context.WithID(r.Context(), uuid.New())
			r = r.WithContext(ctx)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(s.ShortenURL)
			h.ServeHTTP(w, r)

			res := w.Result()
			assert.Equal(t, tc.want.statusCode, res.StatusCode)
			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			url := string(body)
			if res.StatusCode != http.StatusCreated && !strings.HasPrefix(url, tc.want.body) {
				t.Errorf("Expected return body has prefix '%v', got '%v'", tc.want.body, url)
			}
			if tc.saveArg != nil {
				*tc.saveArg = strings.TrimPrefix(url, s.BaseURL+"/")
			}
		})
	}
	// запускаем тесты GET
	for _, tc := range testsGet {
		t.Run(tc.name, func(t *testing.T) {
			path := ""
			if tc.arg != nil {
				path = *tc.arg
			}
			r := httptest.NewRequest("GET", "/"+path, nil)
			r = mux.SetURLVars(r, map[string]string{"id": path}) // чтобы горилла отработала на совесть
			w := httptest.NewRecorder()
			h := http.HandlerFunc(s.DecodeURL)
			h.ServeHTTP(w, r)

			res := w.Result()
			assert.Equal(t, tc.want.statusCode, res.StatusCode)
			defer res.Body.Close()
			b, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			body := strings.TrimSpace(string(b))
			if res.StatusCode == http.StatusTemporaryRedirect {
				location := res.Header.Get("Location")
				assert.Equal(t, tc.want.location, location)
			} else {
				assert.Equal(t, tc.want.body, body)
			}
		})
	}
}

var _ storage.Storage = (*MockStorage)(nil)

type MockStorage struct {
}

func (ms MockStorage) Store(id uuid.UUID, key, url string) error {
	return nil // имитирует сохранение ключа в базе, ошибок быть не может
}

func (ms MockStorage) Get(key string) (string, error) {
	return "", errors.New("Mock error") // Ошибка - элемент не найден (используется в цикле проверки уникальности)
}

func (ms MockStorage) GetAll(id uuid.UUID) map[string]string {
	return nil
}

func TestAPIShorten(t *testing.T) {
	rand.Seed(1)
	const fakeKey = "fpllngzi" // Первый ключ, генерируемый при rand.Seed(1)
	type want struct {
		contentType string
		body        string
		statusCode  int
	}
	testCases := []struct {
		name string
		body string
		want want
	}{
		{
			name: "#1 Valid request",
			body: `{"url" : "http://shetube.com"}`,
			want: want{
				contentType: "application/json",
				body:        `{"result":"http://localhost:8080/` + fakeKey + `"}`,
				statusCode:  http.StatusCreated,
			},
		},
		{
			name: "#2 Invalid url",
			body: `{"url" : "hetube.com"}`,
			want: want{
				contentType: "text/plain; charset=utf-8",
				body:        "Wrong URL",
				statusCode:  http.StatusBadRequest,
			},
		},
		{
			name: "#3 Invalid json",
			body: "{url : http://wetube.com}",
			want: want{
				contentType: "text/plain; charset=utf-8",
				body:        "Bad request",
				statusCode:  http.StatusBadRequest,
			},
		},
	}
	s := shortener.NewShortener("http://localhost:8080", &MockStorage{})
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/api/shorten", strings.NewReader(tc.body))
			ctx := context.WithID(r.Context(), uuid.New())
			r = r.WithContext(ctx)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(s.APIShortenURL)
			h.ServeHTTP(w, r)

			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, tc.want.statusCode, res.StatusCode)
			assert.Equal(t, tc.want.contentType, res.Header.Get("Content-Type"))
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			if res.StatusCode == http.StatusCreated {
				assert.JSONEq(t, tc.want.body, string(body))
			} else {
				assert.Equal(t, tc.want.body, strings.TrimSpace(string(body)))
			}
		})
	}
}
