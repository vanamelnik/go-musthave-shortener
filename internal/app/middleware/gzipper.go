package middleware

import (
	"io"
	"log"
	"net/http"
	"strings"
)

// gzipWriter подменяет собой http.ResponseWriter и сжимает входящие в него данные.
type gzipWriter struct {
	http.ResponseWriter
	// w должен быть установлен как gzipWriter
	w io.Writer
}

func (gw gzipWriter) Write(data []byte) (int, error) {
	log.Printf("gzipWriter: Writting %d bytes...", len(data))

	cnt, err := gw.w.Write(data)
	log.Printf("gzipWriter: Wrote %d bytes, err: %v", cnt, err)
	return cnt, err
}

// Gzipper middleware проверяет, поддерживает ли фронтенд сжатие gzip, и, если да,
// то сжимает тело ответа.
func Gzipper(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		// if err != nil {
		// 	log.Printf("gzipHandle: %v", err)
		// 	http.Error(w, "Internal server error", http.StatusInternalServerError)

		// 	return
		// }
		gz := counter{w: w}

		// w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: w, w: gz}, r)
	})
}

type counter struct {
	w   io.Writer
	cnt int
}

func (c counter) Write(b []byte) (int, error) {
	cnt, err := c.w.Write(b)
	c.cnt = cnt
	return cnt, err
}
