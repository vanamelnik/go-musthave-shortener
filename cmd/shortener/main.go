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

func main() {
	serverAddr, ok := os.LookupEnv("SERVER_ADDRESS")
	if !ok {
		serverAddr = "http://localhost:8080/"
	}
	baseURL, ok := os.LookupEnv("BASE_URL")
	if !ok {
		baseURL = "http://localhost:8080/"
	}
	log.Printf("Server Adress is %s", serverAddr)
	log.Printf("Base URL is %s", baseURL)
	rand.Seed(time.Now().UnixNano())
	db := inmem.NewDB()
	s := shortener.NewShortener(serverAddr, baseURL, db)
	router := mux.NewRouter()
	router.HandleFunc("/{id}", s.DecodeURL).Methods(http.MethodGet)
	router.HandleFunc("/", s.ShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten", s.APIShortenURL).Methods(http.MethodPost)
	server := http.Server{
		Addr:    ":8080",
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
