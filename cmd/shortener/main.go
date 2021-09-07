package main

import (
	"context"
	"flag"
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

	fileNameDefault = "localhost.db"
	baseURLDefault  = "http://localhost:8080"
	srvAddrDefault  = ":8080"
)

type config struct {
	baseURL       string
	srvAddr       string
	fileName      string
	flushInterval time.Duration
}

func main() {
	cfg := getConfig(
		withDefaults(),
		withFlags(),
		withEnv(),
	)
	log.Printf("Server configuration: %+v", *cfg)

	rand.Seed(time.Now().UnixNano())

	db, err := inmem.NewDB(cfg.fileName, cfg.flushInterval)
	if err != nil {
		log.Fatal(err)
	}

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

type fnConf func(*config)

func getConfig(opts ...fnConf) *config {
	cfg := config{}

	for _, fn := range opts {
		fn(&cfg)
	}

	return &cfg
}

func withDefaults() fnConf {
	return func(cfg *config) {
		cfg.baseURL = baseURLDefault
		cfg.srvAddr = srvAddrDefault
		cfg.fileName = fileNameDefault
		cfg.flushInterval = flushInterval
	}
}

func withFlags() fnConf {
	return func(cfg *config) {
		flag.StringVar(&cfg.srvAddr, "a", srvAddrDefault, "Server address")
		flag.StringVar(&cfg.baseURL, "b", baseURLDefault, "Base URL")
		flag.StringVar(&cfg.fileName, "f", fileNameDefault, "File storage path")
		flag.Parse()
	}
}

func withEnv() fnConf {
	return func(cfg *config) {
		env := map[string]*string{
			"BASE_URL":          &cfg.baseURL,
			"SERVER_ADDRESS":    &cfg.srvAddr,
			"FILE_STORAGE_PATH": &cfg.fileName,
		}

		for v := range env {
			if envVal, ok := os.LookupEnv(v); ok {
				*env[v] = envVal
			}
		}
	}
}
