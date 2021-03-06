package grpc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	pb "github.com/vanamelnik/go-musthave-shortener/internal/app/api/grpc/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestPing(t *testing.T) {
	w := startClient(t)
	defer w.conn.Close()

	resp, err := w.client.Ping(context.Background(), &pb.Empty{})
	assert.NoError(t, err)
	assert.Equal(t, true, resp.Ok)
}

func TestShortener(t *testing.T) {
	ctx := context.Background()
	w := startClient(t)
	defer w.conn.Close()

	var shortURL, userID string
	originalURL := "http://yandex.ua"

	t.Run("Shorten single URL", func(t *testing.T) {
		resp, err := w.client.ShortenURL(ctx, &pb.ShortenURLRequest{
			Url: originalURL,
		})
		assert.NoError(t, err)
		assert.Empty(t, resp.Error)
		assert.NotEqual(t, uuid.Nil, resp.UserId)
		shortURL = resp.Result
		userID = resp.UserId
		t.Logf("Successfully get short URL %s for %s, user ID %s", shortURL, originalURL, userID)
	})
	t.Run("Shorten error cases", func(t *testing.T) {
		tt := []struct {
			name   string
			url    string
			userID string
		}{
			{
				name:   "Incorrect URL",
				url:    "на деревню дедушке",
				userID: "",
			},
			{
				name:   "Incorrect user ID",
				url:    "http://google.com",
				userID: "самый правильный UUID",
			},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := w.client.ShortenURL(ctx, &pb.ShortenURLRequest{
					Url:    tc.url,
					UserId: tc.userID,
				})
				assert.NoError(t, err)
				assert.NotEmpty(t, resp.Error)
				assert.Empty(t, resp.Result)
				t.Logf("error message: %s", resp.Error)
			})
		}
	})
	t.Run("Get original URL", func(t *testing.T) {
		resp, err := w.client.DecodeURL(ctx, &pb.DecodeURLRequest{ShortUrl: shortURL})
		assert.NoError(t, err)
		assert.Empty(t, resp.Error)
		assert.Equal(t, originalURL, resp.OriginalUrl)
	})
	t.Run("Get non-existing url", func(t *testing.T) {
		resp, err := w.client.DecodeURL(ctx, &pb.DecodeURLRequest{ShortUrl: "invalidkey"})
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.Error)
		t.Logf("error message: %s", resp.Error)
	})
	t.Run("Stats", func(t *testing.T) {
		resp, err := w.client.Stats(ctx, &pb.Empty{})
		assert.NoError(t, err)
		t.Logf("Stats: urls=%d, users=%d", resp.Urls, resp.Users)
	})
}

func TestBatchShortenAndDelete(t *testing.T) {
	ctx := context.Background()
	w := startClient(t)
	defer w.conn.Close()
	var userID string

	t.Run("BatchShorten 5 URLs", func(t *testing.T) {
		records := []*pb.BatchShortenRequest_Records{
			{
				CorrelationId: "Google",
				Url:           "http://google.com",
			},
			{
				CorrelationId: "Apple",
				Url:           "http://apple.com",
			},
			{
				CorrelationId: "Amazon",
				Url:           "http://amazon.com",
			},
			{
				CorrelationId: "Microsoft",
				Url:           "http://microsoft.com",
			},
			{
				CorrelationId: "Youtube",
				Url:           "http://youtube.com",
			},
		}
		resp, err := w.client.BatchShorten(ctx, &pb.BatchShortenRequest{
			Records: records,
			UserId:  "",
		})
		assert.NoError(t, err)
		userID = resp.UserId
		assert.Equal(t, len(records), len(resp.Records))
		t.Logf("response records: %+v", resp.Records)
	})
	t.Run("Add 1 more URL by the same user", func(t *testing.T) {
		resp, err := w.client.ShortenURL(ctx, &pb.ShortenURLRequest{
			Url:    "http://onemoreurl.io",
			UserId: userID,
		})
		assert.NoError(t, err)
		assert.Empty(t, resp.Error)
	})
	t.Run("List URLs", func(t *testing.T) {
		resp, err := w.client.GetUserURLs(ctx, &pb.GetUserURLsRequest{UserId: userID})
		assert.NoError(t, err)
		assert.Empty(t, resp.Error)
		assert.Equal(t, 6, len(resp.Records)) // 5+1
		t.Logf("URLs list of the user %s: %+v", userID, resp.Records)
	})
	t.Run("List urls for non-existing user", func(t *testing.T) {
		resp, err := w.client.GetUserURLs(ctx, &pb.GetUserURLsRequest{UserId: uuid.NewString()})
		assert.NoError(t, err)
		assert.Empty(t, resp.Error)
		assert.Equal(t, 0, len(resp.Records)) // 5+1
	})
	t.Run("Delete 3 URLs", func(t *testing.T) {
		respGet, err := w.client.GetUserURLs(ctx, &pb.GetUserURLsRequest{UserId: userID})
		assert.NoError(t, err)
		assert.Empty(t, respGet.Error)

		keys := make([]string, 0, 3)
		for i, rec := range respGet.Records {
			if i > 2 {
				break
			}
			keys = append(keys, strings.TrimPrefix(rec.ShortUrl, baseURL+"/"))
		}
		t.Logf("keys to delete: %v", keys)
		resp, err := w.client.DeleteURLs(ctx, &pb.DeleteURLsRequest{
			Keys:   keys,
			UserId: userID,
		})
		assert.NoError(t, err)
		assert.Empty(t, resp.Error)
		time.Sleep(200 * time.Millisecond) // wait when dataloader flushes
		respGet, err = w.client.GetUserURLs(ctx, &pb.GetUserURLsRequest{UserId: userID})
		assert.NoError(t, err)
		assert.Empty(t, respGet.Error)
		assert.Equal(t, 3, len(respGet.Records)) // 6 - 3
		t.Logf("records after deleting: %+v", respGet.Records)
	})
}

type workspace struct {
	conn   *grpc.ClientConn
	client pb.ShortenerClient
}

func startClient(t *testing.T) *workspace {
	conn, err := grpc.Dial(port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	client := pb.NewShortenerClient(conn)
	return &workspace{
		conn:   conn,
		client: client,
	}
}
