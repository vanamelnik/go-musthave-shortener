package rest

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/config"
	"github.com/vanamelnik/go-musthave-shortener/pkg/middleware"
)

// SetupRoutes устанавливает пути для обработчиков ендпоинтов REST API.
func (rest Rest) SetupRoutes(cfg config.Config, router *mux.Router) {
	router.HandleFunc("/ping", rest.Ping).Methods(http.MethodGet)

	router.HandleFunc("/{id}", rest.DecodeURL).Methods(http.MethodGet)
	router.HandleFunc("/", rest.ShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten", rest.APIShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten/batch", rest.BatchShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/user/urls", rest.UserURLs).Methods(http.MethodGet)
	router.HandleFunc("/api/user/urls", rest.DeleteURLs).Methods(http.MethodDelete)

	internal := router.PathPrefix("/api/internal").Subrouter()
	internal.HandleFunc("/stats", rest.Stats).Methods(http.MethodGet)
	internal.Use(middleware.SubnetCheckerMdlw(cfg.TrustedSubnet))

	router.Use(middleware.CookieMdlw(cfg.Secret), middleware.GzipMdlw)
}
