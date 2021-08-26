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

	"github.com/gorilla/mux"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/inmem"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/shortener"
)

const (
	host = "http://localhost"
	port = ":8080"
)

func main() {
	rand.Seed(time.Now().UnixNano()) // ***вопрос*** это лучше делать здесь или каждый раз при генерации ключа?
	db := inmem.NewDB()
	s := shortener.NewShortener(host, port, db)
	router := mux.NewRouter()
	router.HandleFunc("/{id}", s.DecodeURL).Methods(http.MethodGet)
	router.HandleFunc("/", s.ShortenURL).Methods(http.MethodPost)
	server := http.Server{
		Addr:    port,
		Handler: router,
	}
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	// ***вопрос*** или лучше механизм graceful shutdown запустить в горутину?
	go server.ListenAndServe()
	log.Printf("Shortener server is listening at %s...", port)

	<-sigint
	fmt.Print("Shutting down... ")
	server.Shutdown(context.Background())
	fmt.Println("Ok")
}
