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

/*
  APP_PORT=9080
  APP_BASE_HOST=$(tr -dc a-z < /dev/urandom | head -c 12 ; echo '.local')
  APP_BASE_URL=http://$APP_BASE_HOST:$APP_PORT
  echo "APP_PORT=$APP_PORT" >> $GITHUB_ENV
  echo "APP_BASE_HOST=$APP_BASE_HOST" >> $GITHUB_ENV
  echo "APP_BASE_URL=$APP_BASE_URL" >> $GITHUB_ENV
  echo "127.0.0.1 $APP_BASE_HOST" >> /etc/hosts
  bash -c "SERVER_ADDRESS=:$APP_PORT BASE_URL=$APP_BASE_URL FILE_STORAGE_PATH=/tmp/$APP_BASE_HOST.db go run ./cmd/shortener/... &> /dev/null" &
  timeout 10 sh -c "until lsof -i:$APP_PORT; do sleep 1s; done"
*/

type config struct {
	appPort     string
	appBaseHost string
	appBaseUrl  string
	srvAddr     string
}

func main() {
	// env:default
	cfgEnv := map[string]string{
		"APP_PORT":       "8080",
		"APP_BASE_HOST":  "localhost",
		"APP_BASE_URL":   "http://localhost:8080",
		"SERVER_ADDRESS": ":8080",
	}
	cfg := getConfig(cfgEnv)
	log.Printf("Server configuration: %+v", cfg)

	rand.Seed(time.Now().UnixNano())
	db := inmem.NewDB()
	s := shortener.NewShortener(cfg.appBaseUrl, db)

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
	}()
	log.Println("Shortener server is listening at " + cfg.srvAddr)

	<-sigint
	fmt.Print("Shutting down... ")
	if err := server.Shutdown(context.Background()); err != nil {
		log.Println(err)
	}
}

func getConfig(env map[string]string) *config {
	for v := range env {
		if envVal, ok := os.LookupEnv(v); ok {
			env[v] = envVal // изменить значение по умолчанию на значение переменной окружения
		}
	}
	return &config{
		appPort:     env["APP_PORT"],
		appBaseHost: env["APP_BASE_HOST"],
		appBaseUrl:  env["APP_BASE_URL"],
		srvAddr:     env["SERVER_ADDRESS"],
	}
}
