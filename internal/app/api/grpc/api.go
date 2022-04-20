package grpc

import (
	"context"

	pb "github.com/vanamelnik/go-musthave-shortener/internal/app/api/grpc/proto"
	"github.com/vanamelnik/go-musthave-shortener/internal/app/shortener"
	"google.golang.org/grpc"
)

type API struct {
	pb.UnimplementedShortenerServer
	shortener shortener.Shortener
}

func (api API) Ping(ctx context.Context, in *pb.Empty, opts ...grpc.CallOption) (*pb.PingResponse, error) {
	result := true
	if err := api.shortener.Ping(); err != nil {
		result = false
	}
	return &pb.PingResponse{
		Ok: result,
	}, nil
}

func (api API) Stats(ctx context.Context, in *pb.Empty, opts ...grpc.CallOption) (*pb.StatsResponse, error) {
	urls, users, err := api.shortener.Stats(ctx)
	if err != nil {
		return &pb.StatsResponse{
			Error: err.Error(),
		}, nil
	}
	return &pb.StatsResponse{
		Urls:   int32(urls),
		Useres: int32(users),
		Error:  nil,
	}, nil
}
