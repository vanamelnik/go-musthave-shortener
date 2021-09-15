package middleware

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"
)

type gzipReadCloser struct {
	gr   io.ReadCloser
	body io.ReadCloser // вопрос: я так и не понял, закрывает ли gzip.Close() входящий Reader, если он был ReadCloser'ом? Если да, то body здесь не нужен.
}

func (g gzipReadCloser) Read(p []byte) (int, error) {
	return g.gr.Read(p)
}

func (g gzipReadCloser) Close() error {
	if err := g.body.Close(); err != nil {
		return err
	}
	return g.gr.Close()
}

// gzipWriter подменяет собой http.ResponseWriter и сжимает входящие в него данные.
type gzipWriter struct {
	http.ResponseWriter
	// Writer должен быть установлен как gzip.Writer.
	Writer io.Writer
}

func (gw gzipWriter) Write(data []byte) (int, error) {
	return gw.Writer.Write(data)
}

// Gzipper middleware проверяет, поддерживает ли фронтенд сжатие gzip, и, если да,
// то сжимает тело ответа.
func Gzipper(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				log.Printf("gzipHandle: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)

				return
			}

			r.Body = gzipReadCloser{
				gr:   gr,
				body: r.Body,
			}
		}
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gw, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			log.Printf("gzipHandle: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)

			return
		}
		defer gw.Close()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gw}, r)
	})
}
