package handler

import (
	"context"
	"testing"

	"github.com/Den8319/shortener/api/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// grpcCtxWithAuth возвращает gRPC-контекст с метаданными authorization.
func grpcCtxWithAuth(ctx context.Context, token string) context.Context {
	return metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", token))
}

var _ proto.ShortenerServiceServer = (*GRPCHandler)(nil)

func TestGRPCHandler_ShortenURL(t *testing.T) {
	h := NewGRPCHandler(newTestService(t))
	validToken := createTestToken(t, "user-1")

	tests := []struct {
		name     string
		ctx      context.Context
		url      string
		wantErr  bool
		wantCode codes.Code
	}{
		{
			name:     "valid url with auth",
			ctx:      grpcCtxWithAuth(context.Background(), validToken),
			url:      "https://example.com",
			wantErr:  false,
			wantCode: codes.OK,
		},
		{
			name:     "invalid url",
			ctx:      grpcCtxWithAuth(context.Background(), validToken),
			url:      "not-a-url",
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "no auth metadata",
			ctx:      context.Background(),
			url:      "https://example.org",
			wantErr:  true,
			wantCode: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := h.ShortenURL(tt.ctx, &proto.URLShortenRequest{Url: tt.url})
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantCode, status.Code(err))
				assert.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.NotEmpty(t, resp.Result)
		})
	}
}

func TestGRPCHandler_ExpandURL(t *testing.T) {
	h := NewGRPCHandler(newTestService(t))

	// Создаём короткий URL для успешного теста
	ctx := grpcCtxWithAuth(context.Background(), createTestToken(t, "user-1"))
	shortened, err := h.ShortenURL(ctx, &proto.URLShortenRequest{Url: "https://example.com"})
	require.NoError(t, err)

	tests := []struct {
		name     string
		id       string
		wantErr  bool
		wantCode codes.Code
		wantURL  string
	}{
		{
			name:    "existing url",
			id:      shortened.Result,
			wantErr: false,
			wantURL: "https://example.com",
		},
		{
			name:     "nonexistent id",
			id:       "nonexistent",
			wantErr:  true,
			wantCode: codes.NotFound,
		},
		{
			name:     "empty id",
			id:       "",
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := h.ExpandURL(context.Background(), &proto.URLExpandRequest{Id: tt.id})
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantCode, status.Code(err))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.wantURL, resp.Result)
		})
	}
}

func TestGRPCHandler_ListUserURLs(t *testing.T) {
	h := NewGRPCHandler(newTestService(t))

	// Создаём URL от пользователя user-1
	ctxUser1 := grpcCtxWithAuth(context.Background(), createTestToken(t, "user-1"))
	_, err := h.ShortenURL(ctxUser1, &proto.URLShortenRequest{Url: "https://example.com"})
	require.NoError(t, err)

	tests := []struct {
		name       string
		ctx        context.Context
		wantErr    bool
		wantCode   codes.Code
		wantURLCnt int
	}{
		{
			name:       "user-1 has one url",
			ctx:        ctxUser1,
			wantErr:    false,
			wantURLCnt: 1,
		},
		{
			name:       "user-2 has no urls",
			ctx:        grpcCtxWithAuth(context.Background(), createTestToken(t, "user-2")),
			wantErr:    false,
			wantURLCnt: 0,
		},
		{
			name:     "no auth metadata",
			ctx:      context.Background(),
			wantErr:  true,
			wantCode: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := h.ListUserURLs(tt.ctx, &emptypb.Empty{})
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantCode, status.Code(err))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Len(t, resp.Url, tt.wantURLCnt)
		})
	}
}
