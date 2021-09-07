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
	// сперва парсим переменные окружения, т.к. флаги имеют преимущество и
	// в дальнейшем смогут перекрыть их.
	cfgEnv := map[string]string{
		"BASE_URL":          baseURLDefault,
		"SERVER_ADDRESS":    srvAddrDefault,
		"FILE_STORAGE_PATH": fileNameDefault,
	}
	cfg := getConfig(cfgEnv, flushInterval)
	parseFlags(cfg)

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

// getConfig получает на вход map <имя переменной окружения> : <значение по умолчанию>, затем проверяет, не установлена
// ли каждая из этих переменных, и меняет значение на установленную. Функция возвращает структуру config, сформированную
// из скорректированной map.
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

// parseFlags применяет флаги к конфигурации сервиса.
// Если флаги не установлены, в конфигурации сохраняются старые значения
// (заданные переменными окружения либо значения по умолчанию).
func parseFlags(cfg *config) {
	_ = flag.String("a", srvAddrDefault, "Server address")
	_ = flag.String("b", baseURLDefault, "Base URL")
	_ = flag.String("f", fileNameDefault, "File storage path")
	flag.Parse()

	// поскольку дефолтные значения флагов не должны перекрывать переменные окружения,
	// мы добавим в конфигурацию только те флаги, которые пользователь установил явно.
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			cfg.srvAddr = f.Value.String()
		case "b":
			cfg.baseURL = f.Value.String()
		case "f":
			cfg.fileName = f.Value.String()
		}
	})
}
