package shortener_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appContext "github.com/vanamelnik/go-musthave-shortener/internal/app/context"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/dataloader"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/shortener"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/storage/inmem"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/storage/postgres"
)

const (
	baseURL = "http://localhost:8080"
	dsn     = "host=localhost port=5432 user=shortener password=qwe123 dbname=shortener_test"
)

func BenchmarkInmem(b *testing.B) {
	db, err := inmem.NewDB("tmp.db", time.Second)
	require.NoError(b, err)
	defer func() {
		db.Close()
		require.NoError(b, os.Remove("tmp.db"))
	}()
	s := shortener.NewShortener(baseURL, db, dataloader.DataLoader{})
	keys := make([]string, 0, 10000)

	b.Run("Shorten URLs and store them in inmemory storage", shortenBenchmark(s, &keys))
	b.Run("Decode URLs", getRedirectBenchmark(s, keys))
}

func BenchmarkPostgres(b *testing.B) {
	db, err := postgres.NewRepo(context.Background(), dsn)
	require.NoError(b, err)
	defer db.Close()
	dl := dataloader.NewDataLoader(context.Background(), db.BatchDelete, time.Millisecond)
	s := shortener.NewShortener(baseURL, db, dl)
	defer dl.Close()
	keys := make([]string, 0, 10000)
	b.Run("Shorten URLs and store them in postgres storage", shortenBenchmark(s, &keys))
	b.Run("Decode URLs", getRedirectBenchmark(s, keys))

}

func shortenBenchmark(s *shortener.Shortener, keys *[]string) func(b *testing.B) {
	return func(b *testing.B) {
		id := uuid.New()
		for i := 0; i < b.N; i++ {
			url := fmt.Sprintf("http://%s.com", uuid.New().String())
			r := httptest.NewRequest("POST", "/", strings.NewReader(url))
			ctx := appContext.WithID(r.Context(), id)
			r = r.WithContext(ctx)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(s.ShortenURL)
			h.ServeHTTP(w, r)
			res := w.Result()
			key, err := io.ReadAll(res.Body)
			res.Body.Close()
			require.NoError(b, err)
			assert.Equal(b, http.StatusCreated, res.StatusCode)
			*keys = append(*keys, string(key))
		}
		b.Logf("%d urls processed", len(*keys))
	}
}

func getRedirectBenchmark(s *shortener.Shortener, keys []string) func(b *testing.B) {
	return func(b *testing.B) {
		j := 0
		for i := 0; i < b.N; i++ {
			url, err := url.Parse(keys[j])
			require.NoError(b, err)
			r := httptest.NewRequest("GET", url.Path, nil)
			r = mux.SetURLVars(r, map[string]string{"id": strings.TrimPrefix(url.Path, "/")})

			w := httptest.NewRecorder()
			h := http.HandlerFunc(s.DecodeURL)
			h.ServeHTTP(w, r)
			res := w.Result()
			defer res.Body.Close()
			assert.Equal(b, http.StatusTemporaryRedirect, res.StatusCode)
			j++
			if j >= len(keys) {
				j = 0
			}
		}
	}
}
