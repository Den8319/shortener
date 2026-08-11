package handler

import (
	"context"
	"errors"

	"github.com/Den8319/shortener/api/proto"
	"github.com/Den8319/shortener/internal/model"
	"github.com/Den8319/shortener/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GRPCHandler реализует ShortenerService для gRPC-запросов.
type GRPCHandler struct {
	proto.UnimplementedShortenerServiceServer
	service *service.Service
}

// NewGRPCHandler создаёт новый GRPCHandler с указанным сервисом.
func NewGRPCHandler(service *service.Service) *GRPCHandler {
	return &GRPCHandler{service: service}
}

// ShortenURL обрабатывает сокращение URL по gRPC.
func (h *GRPCHandler) ShortenURL(ctx context.Context, req *proto.URLShortenRequest) (*proto.URLShortenResponse, error) {
	userUUID, err := h.service.Authenticate(getAuthTokenFromGRPC(ctx))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid authorization")
	}
	result, err := h.service.Shorten(ctx, req.Url, userUUID)
	if err != nil {
		if errors.Is(err, model.ErrInvalidURL) {
			return nil, status.Error(codes.InvalidArgument, "invalid url")
		}
		return nil, status.Error(codes.Internal, "failed to shorten url")
	}
	return &proto.URLShortenResponse{Result: result.ShortURL}, nil
}

// ExpandURL обрабатывает получение длинного URL по короткому идентификатору.
func (h *GRPCHandler) ExpandURL(ctx context.Context, req *proto.URLExpandRequest) (*proto.URLExpandResponse, error) {
	longURL, err := h.service.Expand(ctx, req.Id)
	if err != nil {
		if errors.Is(err, model.ErrEmptyID) {
			return nil, status.Error(codes.InvalidArgument, "empty id")
		}
		if errors.Is(err, model.ErrURLDeleted) {
			return nil, status.Error(codes.NotFound, "url deleted")
		}
		return nil, status.Error(codes.NotFound, "url not found")
	}
	return &proto.URLExpandResponse{Result: longURL}, nil
}

// ListUserURLs возвращает список URL текущего пользователя.
func (h *GRPCHandler) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*proto.UserURLsResponse, error) {
	userUUID, err := h.service.Authenticate(getAuthTokenFromGRPC(ctx))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid authorization")
	}
	urls, err := h.service.ListUserURLs(ctx, userUUID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user urls")
	}
	result := make([]*proto.URLData, 0, len(urls))
	for _, u := range urls {
		result = append(result, &proto.URLData{
			ShortUrl:    u.ShortURL,
			OriginalUrl: u.LongURL,
		})
	}
	return &proto.UserURLsResponse{Url: result}, nil
}

// getAuthTokenFromGRPC извлекает JWT-токен из gRPC metadata (authorization).
// Возвращает сырую строку токена; валидация выполняется в сервисном слое.
func getAuthTokenFromGRPC(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return ""
	}
	return authHeaders[0]
}
