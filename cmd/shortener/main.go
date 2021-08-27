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
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/shortener"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/storage/inmem"
)

const (
	host = "http://localhost"
	port = ":8080"
)

func main() {
	rand.Seed(time.Now().UnixNano())
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
