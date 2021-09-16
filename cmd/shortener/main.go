package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/middleware"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/shortener"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/storage/inmem"
)

const (
	flushInterval = 10 * time.Second

	secret = "секретный ключ, которым шифруются подписи"

	fileNameDefault = "localhost.db"
	baseURLDefault  = "http://localhost:8080"
	srvAddrDefault  = ":8080"
)

// config определяет базовую конйигурацию сервиса
type config struct {
	baseURL       string
	srvAddr       string
	fileName      string
	flushInterval time.Duration
}

// validate проверяет конфигурацию и выдает ошибку, если обнаруживает пустые поля.
func (cfg config) validate() error {
	problems := make([]string, 0, 4)

	if cfg.baseURL == "" {
		problems = append(problems, "base URL")
	}
	if cfg.srvAddr == "" {
		problems = append(problems, "server address")
	}
	if cfg.fileName == "" {
		problems = append(problems, "storage file name")
	}
	if cfg.flushInterval == 0 {
		problems = append(problems, "flushing interval")
	}

	if len(problems) != 0 {
		errMsg := "Invalid config: " + strings.Join(problems, ", ") + " not set."

		return errors.New(errMsg)
	}

	return nil
}

func main() {
	cfg := newConfig(
		withFlags(),
		withEnv(),
	)
	if err := cfg.validate(); err != nil {
		log.Fatal(err)
	}
	log.Printf("Server configuration: %+v", cfg)

	rand.Seed(time.Now().UnixNano())

	db, err := inmem.NewDB(cfg.fileName, cfg.flushInterval)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	s := shortener.NewShortener(cfg.baseURL, db)

	router := mux.NewRouter()
	router.HandleFunc("/{id}", s.DecodeURL).Methods(http.MethodGet)
	router.HandleFunc("/", s.ShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten", s.APIShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/user/urls", s.UserURLs).Methods(http.MethodGet)

	router.Use(middleware.CookieMdlw(secret), middleware.GzipMdlw)

	server := http.Server{
		Addr:    cfg.srvAddr,
		Handler: router,
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	go func() {
		log.Println(server.ListenAndServe())
	}()
	log.Println("Shortener server is listening at " + cfg.srvAddr)

	<-sigint
	log.Println("Shutting down... ")
	if err := server.Shutdown(context.Background()); err != nil {
		log.Println(err)
	}
}

type configOption func(*config)

// newConfig формирует конфигурацию из значений по умолчанию, затем опционально меняет
// поля при помощи функций configOption.
func newConfig(opts ...configOption) config {
	cfg := config{
		baseURL:       baseURLDefault,
		srvAddr:       srvAddrDefault,
		fileName:      fileNameDefault,
		flushInterval: flushInterval,
	}

	for _, fn := range opts {
		fn(&cfg)
	}

	return cfg
}

func withFlags() configOption {
	return func(cfg *config) {
		flag.StringVar(&cfg.srvAddr, "a", srvAddrDefault, "Server address")
		flag.StringVar(&cfg.baseURL, "b", baseURLDefault, "Base URL")
		flag.StringVar(&cfg.fileName, "f", fileNameDefault, "File storage path")
		flag.Parse()
	}
}

func withEnv() configOption {
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
