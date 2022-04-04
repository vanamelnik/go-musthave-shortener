package main

import (
	"context"
	"encoding/json"
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
	defaultInmemFlushInterval = 10 * time.Second

	defaultDeleteFlushInterval = time.Millisecond

	defaultSecret = "секретный ключ, которым шифруются подписи"

	fileStorageDefault = "localhost.db"
	baseURLDefault     = "http://localhost:80"
	srvAddrDefault     = ":80"

	pprofAddr = ":7070"

	dsnDefault = "host=localhost port=5432 user=postgres password=qwe123 dbname=postgres"

	defaultHost = "go-musthave-shortener.io"

	defaultCfgFileName = "config.json"
)

type setFlags struct {
	configFileName      *string
	baseURL             *string
	srvAddr             *string
	secret              *string
	dbType              *string
	storageFileName     *string
	dsn                 *string
	enableHTTPS         *bool
	deleteFlushInterval *time.Duration
}

// getFlags считывает установленные пользователем флаги и возвращает структуру setFlags.
func getFlags() setFlags {
	var flushInterval int
	configFilename := flag.String("c", defaultCfgFileName, "configuration file")
	srvAddr := flag.String("a", srvAddrDefault, "Server address")
	baseURL := flag.String("b", baseURLDefault, "Base URL")
	secret := flag.String("p", "*****", "Secret key for hashing cookies") // чтобы ключ по умолчанию не отображался в usage, придется действовать из-за угла))
	dbType := flag.String("t", dbInmem, "Storage type (default inmem)\n- inmem\t\tin-memory storage periodically written to .gob file\n"+
		"- postgres\tPostgreSQL database")
	storageFileName := flag.String("f", fileStorageDefault, "File storage path")
	dsn := flag.String("d", "", "Database DSN")
	enableHTTPS := flag.Bool("s", false, "enable HTTPS")
	flag.IntVar(&flushInterval, "F", int(defaultDeleteFlushInterval/time.Millisecond), "Flush interval for accumulate data to delete in milliseconds")
	flag.Parse()

	delFlushInterval := time.Duration(flushInterval) * time.Millisecond

	sf := setFlags{}
	flag.Visit(func(f *flag.Flag) { // установить только те поля структуры setFlags, которые были заданы явно
		switch f.Name {
		case "c":
			sf.configFileName = configFilename
		case "a":
			sf.srvAddr = srvAddr
		case "b":
			sf.baseURL = baseURL
		case "p":
			sf.secret = secret
		case "t":
			sf.dbType = dbType
		case "f":
			sf.storageFileName = storageFileName
		case "d":
			sf.dsn = dsn
		case "s":
			sf.enableHTTPS = enableHTTPS
		case "F":
			sf.deleteFlushInterval = &delFlushInterval
		}
	})

	return sf
}

// Провайдеры хранилища
const (
	dbInmem    = "inmem"
	dbPostgres = "postgres"
)

func main() {
	displayVersionInfo()

	sf := getFlags()
	configFileName, ok := os.LookupEnv("CONFIG")
	if !ok { // если переменная окружения CONFIG не установлена
		if sf.configFileName != nil { // смотрим, не задано ли имя файла конфигурации флагом
			configFileName = *sf.configFileName
		} else {
			configFileName = defaultCfgFileName // если нет, то используем значение по умолчанию
		}
	}
	cfg := newConfig( // порядок имеет значение
		withFile(configFileName),
		withFlags(sf),
		withEnv(), // наивысший приоритет у переменных окружения
	)
	err := cfg.validate()
	if err != nil {
		log.Fatalf("config: %s", err)
	}
	log.Printf("Server configuration: %s", cfg)

	rand.Seed(time.Now().UnixNano())

	var db storage.Storage
	switch cfg.DBType {
	case dbInmem:
		log.Println("Connecting to in-memory storage...")
		db, err = inmem.NewDB(cfg.StorageFileName, cfg.InmemFlushInterval)
	case dbPostgres:
		log.Print("Connecting to Postgres engine...")
		db, err = postgres.NewRepo(context.Background(), cfg.DSN)
	}
	if err != nil {
		log.Fatalf("Connect to db failed: %v", err)
	}
	defer db.Close()

	dl := dataloader.NewDataLoader(context.Background(), db.BatchDelete, cfg.DeleteFlushInterval)
	defer dl.Close()

	s := shortener.NewShortener(cfg.BaseURL, db, dl)
	router := mux.NewRouter()

	router.HandleFunc("/ping", s.Ping).Methods(http.MethodGet)

	router.HandleFunc("/{id}", s.DecodeURL).Methods(http.MethodGet)
	router.HandleFunc("/", s.ShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten", s.APIShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten/batch", s.BatchShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/user/urls", s.UserURLs).Methods(http.MethodGet)
	router.HandleFunc("/api/user/urls", s.DeleteURLs).Methods(http.MethodDelete)

	router.Use(middleware.CookieMdlw(cfg.Secret), middleware.GzipMdlw)

	server := http.Server{
		Addr:    cfg.SrvAddr,
		Handler: router,
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	go func() {
		if !cfg.EnableHTTPS {
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
	log.Println("Shortener server is listening at " + cfg.SrvAddr)
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

// Config определяет базовую конйигурацию сервиса.
type Config struct {
	BaseURL             string        `json:"base_url"`
	SrvAddr             string        `json:"server_address"`
	Secret              string        `json:"secret"`
	DBType              string        `json:"db_type"`
	StorageFileName     string        `json:"file_storage_path"`
	DSN                 string        `json:"database_dsn"`
	EnableHTTPS         bool          `json:"enable_https"`
	InmemFlushInterval  time.Duration `json:"inmem_flush_interval"`
	DeleteFlushInterval time.Duration `json:"delete_flush_interval"`
}

func (cfg Config) String() string {
	var b strings.Builder
	b.WriteString("baseURL='" + cfg.BaseURL + "'")
	b.WriteString(" srvAddr='" + cfg.SrvAddr + "'")
	b.WriteString(" secret='*****'")
	b.WriteString(" dbType='" + cfg.DBType + "'")
	b.WriteString(" deleteFlushInterval=" + cfg.DeleteFlushInterval.String())
	if cfg.StorageFileName != "" {
		b.WriteString(" fileName='" + cfg.StorageFileName + "'")
	}
	if cfg.InmemFlushInterval != 0 {
		b.WriteString(" inmemFlushInterval=" + cfg.InmemFlushInterval.String())
	}
	if cfg.DSN != "" {
		b.WriteString(" dsn='" + cfg.DSN + "'")
	}
	if cfg.EnableHTTPS {
		b.WriteString(" enableHTTPS: yes")
	} else {
		b.WriteString(" enableHTTPS: no")
	}

	return b.String()
}

// validate проверяет конфигурацию и выдает ошибку, если обнаруживает пустые поля.
func (cfg Config) validate() (retErr error) {
	if cfg.BaseURL == "" {
		retErr = multierror.Append(retErr, errors.New("missing base URL"))
	}
	if cfg.SrvAddr == "" {
		retErr = multierror.Append(retErr, errors.New("mising server address"))
	}
	if cfg.DBType != dbInmem && cfg.DBType != dbPostgres {
		retErr = multierror.Append(retErr, errors.New("invalid storage type"))
	}

	return
}

type configOption func(*Config)

// newConfig формирует конфигурацию из значений по умолчанию, затем опционально меняет
// поля при помощи функций configOption.
func newConfig(opts ...configOption) Config {
	cfg := Config{
		BaseURL:             baseURLDefault,
		SrvAddr:             srvAddrDefault,
		StorageFileName:     fileStorageDefault,
		InmemFlushInterval:  defaultInmemFlushInterval,
		DeleteFlushInterval: defaultDeleteFlushInterval,
		DSN:                 "", // значения по умолчанию будут внесены функцией newConfig.
		EnableHTTPS:         false,
	}

	for _, fn := range opts {
		fn(&cfg)
	}

	// Если пользователь передал DSN для Postgres, используем Postgres, игнорируя флаги.
	if cfg.DSN != "" {
		cfg.DBType = dbPostgres
	}

	switch cfg.DBType {
	case dbPostgres:
		if cfg.DSN == "" {
			cfg.DSN = dsnDefault
		}
		cfg.StorageFileName = ""
		cfg.InmemFlushInterval = 0
	case "":
		cfg.DBType = dbInmem
		cfg.DSN = ""
	case dbInmem:
		cfg.DSN = ""
	}

	return cfg
}

func withFile(filename string) configOption {
	return func(cfg *Config) {
		log.Printf("open file %s", filename)
		f, err := os.Open(filename)
		if err != nil {
			log.Fatalf("config: could not read file %s: %s", filename, err)
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&cfg); err != nil {
			log.Fatalf("config: could not decode file %s: %s", filename, err)
		}
	}
}

// withFlags устанавливает значения конфигурации в соответствии с переданными флагами.
func withFlags(sf setFlags) configOption {
	return func(cfg *Config) {
		if sf.baseURL != nil {
			cfg.BaseURL = *sf.baseURL
		}
		if sf.srvAddr != nil {
			cfg.SrvAddr = *sf.srvAddr
		}
		if sf.secret != nil {
			cfg.Secret = *sf.secret
		}
		if sf.dbType != nil {
			cfg.DBType = *sf.dbType
		}
		if sf.storageFileName != nil {
			cfg.StorageFileName = *sf.storageFileName
		}
		if sf.dsn != nil {
			cfg.DSN = *sf.dsn
		}
		if sf.enableHTTPS != nil {
			cfg.EnableHTTPS = *sf.enableHTTPS
		}
		if sf.deleteFlushInterval != nil {
			cfg.DeleteFlushInterval = *sf.deleteFlushInterval
		}
	}
}

func withEnv() configOption {
	return func(cfg *Config) {
		env := map[string]*string{
			"BASE_URL":          &cfg.BaseURL,
			"SERVER_ADDRESS":    &cfg.SrvAddr,
			"FILE_STORAGE_PATH": &cfg.StorageFileName,
			"DATABASE_DSN":      &cfg.DSN,
			"HASH_KEY":          &cfg.Secret,
		}

		for v := range env {
			if envVal, ok := os.LookupEnv(v); ok {
				*env[v] = envVal
			}
		}
		if _, ok := os.LookupEnv("ENABLE_HTTPS"); ok {
			cfg.EnableHTTPS = true
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
