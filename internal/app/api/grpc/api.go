package grpc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	pb "github.com/vanamelnik/go-musthave-shortener/internal/app/api/grpc/proto"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/middleware"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/shortener"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/storage"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedShortenerServer
	shortener *shortener.Shortener
}

func NewServer(shortener *shortener.Shortener) *grpc.Server {
	s := grpc.NewServer()
	pb.RegisterShortenerServer(s, &server{shortener: shortener})
	return s
}

func (s server) Ping(ctx context.Context, in *pb.Empty) (*pb.PingResponse, error) {
	result := true
	if err := s.shortener.Ping(); err != nil {
		result = false
	}
	return &pb.PingResponse{
		Ok: result,
	}, nil
}

func (s server) ShortenURL(ctx context.Context, r *pb.ShortenURLRequest) (*pb.ShortenURLResponse, error) {
	resp := pb.ShortenURLResponse{}
	id, errStr := getUserID(r.UserId)
	if errStr != "" {
		return &pb.ShortenURLResponse{Error: errStr}, nil
	}
	resp.UserId = id.String()
	shortURL, err := s.shortener.ShortenURL(ctx, id, r.Url)
	if err != nil {
		var errURLAlreadyExists *storage.ErrURLArlreadyExists
		if errors.As(err, &errURLAlreadyExists) {
			shortURL = fmt.Sprintf("%s/%s", s.shortener.BaseURL, errURLAlreadyExists.Key)
		} else {
			return &pb.ShortenURLResponse{Error: err.Error()}, nil
		}
	}
	resp.Result = shortURL
	return &resp, nil
}
func (s server) DecodeURL(ctx context.Context, r *pb.DecodeURLRequest) (*pb.DecodeURLResqponse, error) {
	key := strings.TrimPrefix(r.ShortUrl, s.shortener.BaseURL+"/")
	url, err := s.shortener.DecodeURL(ctx, key)
	if err != nil {
		log.Printf("gRPC: DecodeURL: %s", err)
		return &pb.DecodeURLResqponse{Error: err.Error()}, nil
	}
	return &pb.DecodeURLResqponse{
		OriginalUrl: url,
		Error:       "",
	}, nil
}

func (s server) BatchShorten(ctx context.Context, r *pb.BatchShortenRequest) (*pb.BatchShortenResponse, error) {
	if len(r.Records) == 0 {
		return &pb.BatchShortenResponse{}, nil
	}
	resp := pb.BatchShortenResponse{}
	id, errStr := getUserID(r.UserId)
	if errStr != "" {
		return &pb.BatchShortenResponse{Error: errStr}, nil
	}
	resp.UserId = id.String()
	reqRecords := make([]shortener.BatchShortenRequest, len(r.Records))
	for i, rec := range r.Records {
		reqRecords[i].CorrelationID = rec.CorrelationId
		reqRecords[i].OriginalURL = rec.Url
	}
	result, err := s.shortener.BatchShortenURL(ctx, id, reqRecords)
	if err != nil {
		log.Printf("gRPC: BatchShorten: %s", err)
		return &pb.BatchShortenResponse{Error: err.Error()}, nil
	}
	respRecords := make([]*pb.BatchShortenResponse_Records, len(result))
	for i, rec := range result {
		respRecords[i] = &pb.BatchShortenResponse_Records{
			CorrelationId: rec.CorrelationID,
			ShortUrl:      rec.ShortURL,
		}
	}
	resp.Records = respRecords
	return &resp, nil
}

func (s server) GetUserURLs(ctx context.Context, r *pb.GetUserURLsRequest) (*pb.GetUserURLsResponse, error) {
	id, err := uuid.Parse(r.UserId)
	if err != nil {
		log.Printf("gRPC: GetUserURLs: %s", err)
		return &pb.GetUserURLsResponse{Error: err.Error()}, nil
	}
	result := s.shortener.GetAll(ctx, id)
	records := make([]*pb.GetUserURLsResponse_Record, 0, len(result))
	for key, url := range result {
		records = append(records, &pb.GetUserURLsResponse_Record{
			ShortUrl:    fmt.Sprintf("%s/%s", s.shortener.BaseURL, key),
			OriginalUrl: url,
		})
	}
	return &pb.GetUserURLsResponse{
		Records: records,
		Error:   "",
	}, nil
}

func (s server) Stats(ctx context.Context, in *pb.Empty) (*pb.StatsResponse, error) {
	urls, users, err := s.shortener.Stats(ctx)
	if err != nil {
		return &pb.StatsResponse{
			Error: err.Error(),
		}, nil
	}
	return &pb.StatsResponse{
		Urls:  int32(urls),
		Users: int32(users),
		Error: "",
	}, nil
}

func getUserID(reqUserID string) (uuid.UUID, string) {
	var err error
	var id uuid.UUID
	if reqUserID != "" {
		id, err = uuid.Parse(reqUserID)
		if err != nil {
			log.Printf("gRPC: ShortenURL: could not parse uuid %s: %s", reqUserID, err)
			return uuid.Nil, respWrongID
		}

		return id, ""
	}
	id, err = middleware.GenerateUserID()
	if err != nil {
		log.Printf("gRPC: ShortenURL: could not generate uuid: %s", err)
		return uuid.Nil, respInternalServerError
	}

	return id, ""
}
