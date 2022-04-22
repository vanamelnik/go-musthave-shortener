package rest

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/config"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/middleware"
)

// SetupRoutes устанавливает пути для обработчиков ендпоинтов REST API.
func (api Rest) SetupRoutes(cfg config.Config, router *mux.Router) {
	router.HandleFunc("/ping", api.Ping).Methods(http.MethodGet)

	router.HandleFunc("/{id}", api.DecodeURL).Methods(http.MethodGet)
	router.HandleFunc("/", api.ShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten", api.APIShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten/batch", api.BatchShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/user/urls", api.UserURLs).Methods(http.MethodGet)
	router.HandleFunc("/api/user/urls", api.DeleteURLs).Methods(http.MethodDelete)

	internal := router.PathPrefix("/api/internal").Subrouter()
	internal.HandleFunc("/stats", api.Stats).Methods(http.MethodGet)
	internal.Use(middleware.SubnetCheckerMdlw(cfg.TrustedSubnet))

	router.Use(middleware.CookieMdlw(cfg.Secret), middleware.GzipMdlw)
}
