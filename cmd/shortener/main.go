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

	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app"
)

var shortener *app.Shortener

// controller перенаправляет запрос '/' на обработчика в зависимости от метода
func controller(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		shortener.DecodeURL(w, r)
	case http.MethodPost:
		shortener.ShortenURL(w, r)
	default:
		log.Printf("controller: method not allowed: %v", r.Method)
		http.Error(w, "Wrong method", http.StatusMethodNotAllowed)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano()) // ***вопрос*** это лучше делать здесь или каждый раз при генерации ключа?
	shortener = app.NewShortener()
	server := http.Server{
		Addr: ":8080",
	}
	http.HandleFunc("/", controller)
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	// ***вопрос*** или лучше механизм graceful shutdown запустить в горутину?
	go server.ListenAndServe()
	log.Println("Shortener server is listening at :8080...")

	<-sigint
	fmt.Print("Shutting down... ")
	server.Shutdown(context.Background())
	fmt.Println("Ok")
}
