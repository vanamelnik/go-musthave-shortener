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
	body io.ReadCloser
}

func (g gzipReadCloser) Read(p []byte) (int, error) {
	return g.gr.Read(p)
}

func (g gzipReadCloser) Close() error {
	if err := g.gr.Close(); err != nil {
		return err
	}
	return g.body.Close()
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

// GzipMdlw распаковывет тело запроса при наличии сжатия и проверяет, поддерживает ли фронтенд сжатие ответа gzip,
// и, если да, то сжимает тело ответа.
func GzipMdlw(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Переопределим request, если нужна распаковка
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				log.Printf("gzipHandle: %v", err)
				http.Error(w, "Something went wrong", http.StatusInternalServerError)

				return
			}

			r.Body = gzipReadCloser{
				gr:   gr,
				body: r.Body,
			}
		}

		// Переопределим response writer, если нужна запаковка
		respWriter := w
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			gw, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				log.Printf("gzipHandle: %v", err)
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
				return
			}
			defer gw.Close()

			respWriter = gzipWriter{ResponseWriter: w, Writer: gw}
			respWriter.Header().Set("Content-Encoding", "gzip")
		}

		next.ServeHTTP(respWriter, r)
	})
}
