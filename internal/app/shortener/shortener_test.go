package shortener_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/inmem"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/shortener"
)

func TestShortener(t *testing.T) {
	type want struct {
		statusCode int
		body       string
		location   string
	}
	// Аргументы для GET тестов. Первые два поля заполняются POST тестами
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
				body:       "Wrong url",
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
			name: "Test GET#4 - fake key (not stored in repo)",
			arg:  &args[2],
			want: want{
				statusCode: http.StatusNotFound, // ***вопрос - такой должен быть статус?
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
	db := inmem.NewDB()
	s := shortener.NewShortener(db)
	// запускаем тесты POST
	for _, tc := range testsPost {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			w := httptest.NewRecorder()
			h := http.HandlerFunc(s.ShortenURL)
			h.ServeHTTP(w, r)

			res := w.Result()
			if res.StatusCode != tc.want.statusCode {
				t.Errorf("Expected status code %v, got %v", tc.want.statusCode, res.StatusCode)
			}
			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			url := string(body)
			if res.StatusCode != http.StatusCreated && !strings.HasPrefix(url, tc.want.body) {
				t.Errorf("Expected return body has prefix '%v', got '%v'", tc.want.body, url)
			}
			if tc.saveArg != nil {
				*tc.saveArg = strings.TrimPrefix(url, "http://localhost:8080/")
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
			w := httptest.NewRecorder()
			h := http.HandlerFunc(s.DecodeURL)
			h.ServeHTTP(w, r)

			res := w.Result()
			if res.StatusCode != tc.want.statusCode {
				t.Errorf("Expected status code %v, got %v", tc.want.statusCode, res.StatusCode)
			}
			defer res.Body.Close()
			b, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			body := strings.TrimSpace(string(b))
			if res.StatusCode == http.StatusTemporaryRedirect {
				location := res.Header.Get("Location")
				if location != tc.want.location {
					t.Errorf("Expected location %v, got %v", tc.want.location, location)
				}
			} else {
				if body != tc.want.body {
					t.Errorf("Expected body '%v', got '%v'", tc.want.body, body)
				}
			}
		})
	}
}
