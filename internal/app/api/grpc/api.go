package grpc

import (
	"context"

	pb "github.com/vanamelnik/go-musthave-shortener/internal/app/api/grpc/proto"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/shortener"
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

func (grpc server) Ping(ctx context.Context, in *pb.Empty) (*pb.PingResponse, error) {
	result := true
	if err := grpc.shortener.Ping(); err != nil {
		result = false
	}
	return &pb.PingResponse{
		Ok: result,
	}, nil
}

func (grpc server) Stats(ctx context.Context, in *pb.Empty) (*pb.StatsResponse, error) {
	urls, users, err := grpc.shortener.Stats(ctx)
	if err != nil {
		return &pb.StatsResponse{
			Error: err.Error(),
		}, nil
	}
	return &pb.StatsResponse{
		Urls:   int32(urls),
		Useres: int32(users),
		Error:  "",
	}, nil
}
