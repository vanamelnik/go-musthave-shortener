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
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/storage"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/storage/inmem"
	"github.com/vanamelnik/go-musthave-shortener-tpl/internal/app/storage/postgres"
)

const (
	flushInterval = 10 * time.Second

	secret = "секретный ключ, которым шифруются подписи"

	fileNameDefault = "localhost.db"
	baseURLDefault  = "http://localhost:8080"
	srvAddrDefault  = ":8080"

	dsnDefault = "host=localhost port=5432 user=postgres password=qwe123 dbname=postgres"
)

// config определяет базовую конйигурацию сервиса
type config struct {
	baseURL       string
	srvAddr       string
	fileName      string
	flushInterval time.Duration
	dsn           string
	inMemory      bool
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
	if cfg.fileName == "" && cfg.inMemory {
		problems = append(problems, "storage file name")
	}
	if cfg.flushInterval == 0 && cfg.inMemory {
		problems = append(problems, "flushing interval")
	}
	if cfg.dsn == "" && !cfg.inMemory {
		problems = append(problems, "DSN")
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
	err := cfg.validate()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Server configuration: %+v", cfg)

	rand.Seed(time.Now().UnixNano())

	var db storage.Storage
	if cfg.inMemory {
		log.Println("Connecting to in-memory storage...")
		db, err = inmem.NewDB(cfg.fileName, cfg.flushInterval)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
	} else {
		log.Println("Connecting to Postgres engine...")
		db, err = postgres.NewRepo(cfg.dsn)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
	}

	s := shortener.NewShortener(cfg.baseURL, db)

	router := mux.NewRouter()

	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			log.Printf("postgres: ping: %v", err)
			http.Error(w, "Something went wrong", http.StatusInternalServerError)

			return
		}
		log.Println("postgres: ping OK")
	}).Methods(http.MethodGet)

	router.HandleFunc("/{id}", s.DecodeURL).Methods(http.MethodGet)
	router.HandleFunc("/", s.ShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten", s.APIShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten/batch", s.BatchShortenURL).Methods(http.MethodPost)
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
		inMemory:      true,
		fileName:      fileNameDefault,
		flushInterval: flushInterval,
		dsn:           "", // значения по умолчанию будут внесены функцией newConfig.
	}

	for _, fn := range opts {
		fn(&cfg)
	}

	// Если пользователь передал DSN для Postgres, используем Postgres
	if cfg.dsn != "" {
		cfg.inMemory = false
	}

	if !cfg.inMemory {
		// если пользователь указал использование postgres, но не передал данных DSN, используются DSN по умолчанию.
		if cfg.dsn == "" {
			cfg.dsn = dsnDefault
		}
		// обнуляем поля конфигурации, не нужные для работы postgres
		cfg.fileName = ""
		cfg.flushInterval = 0
	}

	return cfg
}

func withFlags() configOption {
	return func(cfg *config) {
		flag.StringVar(&cfg.srvAddr, "a", srvAddrDefault, "Server address")
		flag.StringVar(&cfg.baseURL, "b", baseURLDefault, "Base URL")
		flag.StringVar(&cfg.fileName, "f", fileNameDefault, "File storage path")
		flag.StringVar(&cfg.dsn, "d", "", "Database DSN")
		flag.BoolVar(&cfg.inMemory, "i", true, "Use in-memory repository instead of postgres DB.")
		flag.Parse()
	}
}

func withEnv() configOption {
	return func(cfg *config) {
		env := map[string]*string{
			"BASE_URL":          &cfg.baseURL,
			"SERVER_ADDRESS":    &cfg.srvAddr,
			"FILE_STORAGE_PATH": &cfg.fileName,
			"DATABASE_DSN":      &cfg.dsn,
		}

		for v := range env {
			if envVal, ok := os.LookupEnv(v); ok {
				*env[v] = envVal
			}
		}
	}
}
