package grpc

import (
	"context"
	"log"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"

	"github.com/vanamelnik/go-musthave-shortener/internal/app/dataloader"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/shortener"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/storage/inmem"
)

const (
	baseURL   = "http://localhost:8080"
	tmpDbFile = "tmpdb.gob"
	port      = ":3200"
)

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	db, err := inmem.NewDB(tmpDbFile, time.Millisecond)
	if err != nil {
		log.Fatal(err)
	}
	// в конце удаляем временный файл базы данных
	defer func() {
		if err := os.Remove(tmpDbFile); err != nil {
			log.Fatal(err)
		}
	}()
	defer db.Close()
	dl := dataloader.NewDataLoader(context.Background(), db.BatchDelete, time.Millisecond)
	defer dl.Close()
	s := shortener.NewShortener(baseURL, db, dl)

	server := NewServer(s)
	listen, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatal(err)
	}
	// запускаем gRPC сервер
	go func() {
		if err := server.Serve(listen); err != nil {
			log.Printf("gRPC server: %s", err)
		}
	}()
	m.Run()
}
