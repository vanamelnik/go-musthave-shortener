package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/gorilla/mux"
	grpc_api "github.com/vanamelnik/go-musthave-shortener/internal/app/api/grpc"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/api/rest"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/config"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/dataloader"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/shortener"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/storage"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/storage/inmem"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/storage/postgres"
	"golang.org/x/crypto/acme/autocert"
)

const (
	// заглушка для пробного запуска HTTPS-сервера.
	defaultHost = "go-musthave-shortener.io"
)

// Информация о версии
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	displayVersionInfo()

	flags := config.GetFlags()
	configFileName, ok := os.LookupEnv("CONFIG")
	if !ok { // если переменная окружения CONFIG не установлена
		if flags.ConfigFileName != nil { // смотрим, не задано ли имя файла конфигурации флагом
			configFileName = *flags.ConfigFileName
		} else {
			configFileName = config.DefaultCfgFileName // если нет, то используем значение по умолчанию
		}
	}
	cfg := config.NewConfig( // порядок имеет значение
		config.WithFile(configFileName),
		config.WithFlags(flags),
		config.WithEnv(), // наивысший приоритет у переменных окружения
	)
	err := cfg.Validate()
	if err != nil {
		log.Fatalf("config: %s", err)
	}
	log.Printf("Server configuration: %s", cfg)

	rand.Seed(time.Now().UnixNano())

	var db storage.Storage
	switch cfg.DBType {
	case config.DBInmem:
		log.Println("Connecting to in-memory storage...")
		db, err = inmem.NewDB(cfg.StorageFileName, cfg.InmemFlushInterval)
	case config.DBPostgres:
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
	rest := rest.NewRest(s)
	rest.SetupRoutes(cfg, router)

	server := http.Server{
		Addr:    cfg.SrvAddr,
		Handler: router,
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	go runMainServer(server, cfg)
	log.Println("Shortener server is listening at " + cfg.SrvAddr)

	go runPprofServer(cfg.PprofAddress)

	go runGRPCServer(cfg.GRPCPort, s)

	<-sigint
	log.Println("Shutting down... ")
	if err := server.Shutdown(context.Background()); err != nil {
		log.Println(err)
	}
}

func runMainServer(server http.Server, cfg config.Config) {
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
}

func runPprofServer(pprofAddress string) {
	if pprofAddress == "" {
		return
	}
	log.Printf("pprof server is listening at %s", pprofAddress)
	err := http.ListenAndServe(pprofAddress, nil)
	if err != nil {
		log.Printf("pprof server: %s", err)
	}
}

func runGRPCServer(gRPCPort string, s *shortener.Shortener) {
	if gRPCPort == "" {
		return
	}
	server := grpc_api.NewServer(s)
	listen, err := net.Listen("tcp", gRPCPort)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("gRPC server is listening at %s", gRPCPort)
	if err := server.Serve(listen); err != nil {
		log.Printf("gRPC server: %s", err)
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
