package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/shortener"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/storage/inmem"
)

// controller перенаправляет запрос '/' на обработчика в зависимости от метода.
func controller(s *shortener.Shortener) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			s.DecodeURL(w, r)
		case http.MethodPost:
			s.ShortenURL(w, r)
		default:
			log.Printf("controller: method not allowed: %v", r.Method)
			http.Error(w, "Wrong method", http.StatusMethodNotAllowed)
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	db := inmem.NewDB()
	s := shortener.NewShortener(db)
	server := http.Server{
		Addr: ":8080",
	}

	http.HandleFunc("/", controller(s))
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	go func() {
		log.Println(server.ListenAndServe())
	}()
	log.Println("Shortener server is listening at :8080...")

	<-sigint
	fmt.Print("Shutting down... ")
	if err := server.Shutdown(context.Background()); err != nil {
		log.Println(err)
	}
}
