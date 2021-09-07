package main

import (
	"context"
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
	flushInterval = 10 * time.Second
	fileName      = "localhost.gob"
)

type config struct {
	baseURL       string
	srvAddr       string
	fileName      string
	flushInterval time.Duration
}

func main() {
	cfgEnv := map[string]string{
		"BASE_URL":          "http://localhost:8080",
		"SERVER_ADDRESS":    ":8080",
		"FILE_STORAGE_PATH": "./" + fileName,
	}
	cfg := getConfig(cfgEnv, flushInterval)
	log.Printf("Server configuration: %+v", *cfg)

	rand.Seed(time.Now().UnixNano())

	db := inmem.NewDB(cfg.fileName, cfg.flushInterval)

	s := shortener.NewShortener(cfg.baseURL, db)

	router := mux.NewRouter()
	router.HandleFunc("/{id}", s.DecodeURL).Methods(http.MethodGet)
	router.HandleFunc("/", s.ShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten", s.APIShortenURL).Methods(http.MethodPost)

	server := http.Server{
		Addr:    cfg.srvAddr,
		Handler: router,
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	go func() {
		log.Println(server.ListenAndServe())
		db.Close() // вопрос: через defer не получается сохранить данные (почему-то) - такой вариант приемлем?
	}()
	log.Println("Shortener server is listening at " + cfg.srvAddr)

	<-sigint
	log.Println("Shutting down... ")
	if err := server.Shutdown(context.Background()); err != nil {
		log.Println(err)
	}
}

func getConfig(env map[string]string, flushInterval time.Duration) *config {
	for v := range env {
		if envVal, ok := os.LookupEnv(v); ok {
			env[v] = envVal // изменить значение по умолчанию на значение переменной окружения
		}
	}
	return &config{
		baseURL:       env["BASE_URL"],
		srvAddr:       env["SERVER_ADDRESS"],
		fileName:      env["FILE_STORAGE_PATH"],
		flushInterval: flushInterval,
	}
}
