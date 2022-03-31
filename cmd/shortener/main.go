package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	_ "net/http/pprof"

	"github.com/gorilla/mux"
	"github.com/hashicorp/go-multierror"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/dataloader"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/middleware"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/shortener"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/storage"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/storage/inmem"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/storage/postgres"
	"golang.org/x/crypto/acme/autocert"
)

// Информация о версии
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// Значения по умолчанию
const (
	inmemFlushInterval = 10 * time.Second

	deleteFlushInterval = time.Millisecond

	defaultSecret = "секретный ключ, которым шифруются подписи"

	fileNameDefault = "localhost.db"
	baseURLDefault  = "http://localhost:8080"
	srvAddrDefault  = ":8080"

	pprofAddr = ":7070"

	dsnDefault = "host=localhost port=5432 user=postgres password=qwe123 dbname=postgres"

	defaultHost = "go-musthave-shortener.io"
)

// Провайдеры хранилища
const (
	dbInmem    = "inmem"
	dbPostgres = "postgres"
)

// config определяет базовую конйигурацию сервиса.
type config struct {
	baseURL string
	srvAddr string
	secret  string

	dbType string

	fileName           string
	inmemFlushInterval time.Duration

	deleteFlushInterval time.Duration

	dsn string

	enableHTTPS bool
}

func (cfg config) String() string {
	var b strings.Builder
	b.WriteString("baseURL='" + cfg.baseURL + "'")
	b.WriteString(" srvAddr='" + cfg.srvAddr + "'")
	b.WriteString(" secret='*****'")
	b.WriteString(" dbType='" + cfg.dbType + "'")
	b.WriteString(" deleteFlushInterval=" + cfg.deleteFlushInterval.String())
	if cfg.fileName != "" {
		b.WriteString(" fileName='" + cfg.fileName + "'")
	}
	if cfg.inmemFlushInterval != 0 {
		b.WriteString(" inmemFlushInterval=" + cfg.inmemFlushInterval.String())
	}
	if cfg.dsn != "" {
		b.WriteString(" dsn='" + cfg.dsn + "'")
	}
	if cfg.enableHTTPS {
		b.WriteString(" enableHTTPS: yes")
	} else {
		b.WriteString(" enableHTTPS: no")
	}

	return b.String()
}

// validate проверяет конфигурацию и выдает ошибку, если обнаруживает пустые поля.
func (cfg config) validate() (retErr error) {
	if cfg.baseURL == "" {
		retErr = multierror.Append(retErr, errors.New("missing base URL"))
	}
	if cfg.srvAddr == "" {
		retErr = multierror.Append(retErr, errors.New("mising server address"))
	}
	if cfg.dbType != dbInmem && cfg.dbType != dbPostgres {
		retErr = multierror.Append(retErr, errors.New("invalid storage type"))
	}

	return
}

func main() {
	displayVersionInfo()
	cfg := newConfig(
		withFlags(),
		withEnv(),
	)
	err := cfg.validate()
	if err != nil {
		log.Fatalf("config: %s", err)
	}
	log.Printf("Server configuration: %s", cfg)

	rand.Seed(time.Now().UnixNano())

	var db storage.Storage
	switch cfg.dbType {
	case dbInmem:
		log.Println("Connecting to in-memory storage...")
		db, err = inmem.NewDB(cfg.fileName, cfg.inmemFlushInterval)
	case dbPostgres:
		log.Print("Connecting to Postgres engine...")
		db, err = postgres.NewRepo(context.Background(), cfg.dsn)
	}
	if err != nil {
		log.Fatalf("Connect to db failed: %v", err)
	}
	defer db.Close()

	dl := dataloader.NewDataLoader(context.Background(), db.BatchDelete, cfg.deleteFlushInterval)
	defer dl.Close()

	s := shortener.NewShortener(cfg.baseURL, db, dl)
	router := mux.NewRouter()

	router.HandleFunc("/ping", s.Ping).Methods(http.MethodGet)

	router.HandleFunc("/{id}", s.DecodeURL).Methods(http.MethodGet)
	router.HandleFunc("/", s.ShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten", s.APIShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten/batch", s.BatchShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/user/urls", s.UserURLs).Methods(http.MethodGet)
	router.HandleFunc("/api/user/urls", s.DeleteURLs).Methods(http.MethodDelete)

	router.Use(middleware.CookieMdlw(cfg.secret), middleware.GzipMdlw)

	server := http.Server{
		Addr:    cfg.srvAddr,
		Handler: router,
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	go func() {
		if !cfg.enableHTTPS {
			log.Println(server.ListenAndServe())
			return
		}
		manager := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			Cache:      autocert.DirCache("cache-dir"),
			HostPolicy: autocert.HostWhitelist(defaultHost, "www."+defaultHost),
		}
		server.TLSConfig = manager.TLSConfig()
		log.Println(server.ListenAndServeTLS("", ""))
	}()
	log.Println("Shortener server is listening at " + cfg.srvAddr)
	go func() {
		log.Println(http.ListenAndServe(pprofAddr, nil))
	}()
	log.Printf("pprof server listening at %s", pprofAddr)
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
		baseURL:             baseURLDefault,
		srvAddr:             srvAddrDefault,
		fileName:            fileNameDefault,
		inmemFlushInterval:  inmemFlushInterval,
		deleteFlushInterval: deleteFlushInterval,
		dsn:                 "", // значения по умолчанию будут внесены функцией newConfig.
		enableHTTPS:         false,
	}

	for _, fn := range opts {
		fn(&cfg)
	}

	// Если пользователь передал DSN для Postgres, используем Postgres, игнорируя флаги.
	if cfg.dsn != "" {
		cfg.dbType = dbPostgres
	}

	switch cfg.dbType {
	case dbPostgres:
		if cfg.dsn == "" {
			cfg.dsn = dsnDefault
		}
		cfg.fileName = ""
		cfg.inmemFlushInterval = 0
	case dbInmem:
		cfg.dsn = ""
	}

	return cfg
}

func withFlags() configOption {
	return func(cfg *config) {
		var flInt int
		flag.StringVar(&cfg.srvAddr, "a", srvAddrDefault, "Server address")
		flag.StringVar(&cfg.baseURL, "b", baseURLDefault, "Base URL")
		flag.StringVar(&cfg.secret, "p", "*****", "Secret key for hashing cookies") // чтобы ключ по умолчанию не отображался в usage, придется действовать из-за угла))
		flag.StringVar(&cfg.dbType, "t", dbInmem, "Storage type (default inmem)\n- inmem\t\tin-memory storage periodically written to .gob file\n"+
			"- postgres\tPostgreSQL database")
		flag.StringVar(&cfg.fileName, "f", fileNameDefault, "File storage path")
		flag.StringVar(&cfg.dsn, "d", "", "Database DSN")
		flag.IntVar(&flInt, "F", int(deleteFlushInterval/time.Millisecond), "Flush interval for accumulate data to delete in milliseconds")
		flag.BoolVar(&cfg.enableHTTPS, "s", false, "enable HTTPS")
		flag.Parse()

		cfg.deleteFlushInterval = time.Duration(flInt) * time.Millisecond
		setByUser := false
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "p" {
				setByUser = true
			}
		})
		if !setByUser {
			cfg.secret = defaultSecret
		}
	}
}

func withEnv() configOption {
	return func(cfg *config) {
		env := map[string]*string{
			"BASE_URL":          &cfg.baseURL,
			"SERVER_ADDRESS":    &cfg.srvAddr,
			"FILE_STORAGE_PATH": &cfg.fileName,
			"DATABASE_DSN":      &cfg.dsn,
			"HASH_KEY":          &cfg.secret,
		}

		for v := range env {
			if envVal, ok := os.LookupEnv(v); ok {
				*env[v] = envVal
			}
		}
		if _, ok := os.LookupEnv("ENABLE_HTTPS"); ok {
			cfg.enableHTTPS = true
		}
	}
}

func displayVersionInfo() {
	if buildVersion == "" {
		buildVersion = "N/A"
	}
	if buildDate == "" {
		buildDate = "N/A"
	}
	if buildCommit == "" {
		buildCommit = "N/A"
	}
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
	fmt.Println()
}
