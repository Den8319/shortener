package handler

import (
	"context"
	"errors"

	"github.com/Den8319/shortener/api/proto"
	"github.com/Den8319/shortener/internal/auth"
	"github.com/Den8319/shortener/internal/model"
	"github.com/rs/zerolog/log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GRPCHandler реализует ShortenerService для gRPC-запросов.
type GRPCHandler struct {
	proto.UnimplementedShortenerServiceServer
	store model.Storage
}

// NewGRPCHandler создаёт новый GRPCHandler с указанным хранилищем.
func NewGRPCHandler(store model.Storage) *GRPCHandler {
	return &GRPCHandler{store: store}
}

// ShortenURL обрабатывает сокращение URL по gRPC.
func (h *GRPCHandler) ShortenURL(ctx context.Context, req *proto.URLShortenRequest) (*proto.URLShortenResponse, error) {

	if !isValidURL(req.Url) {
		return nil, status.Error(codes.InvalidArgument, "invalid url")
	}

	userUUID := getUserFromGRPC(ctx)
	if userUUID == "" {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid authorization")
	}

	shortURL, err := h.store.GetShortURL(ctx, req.Url, userUUID)
	if err != nil {
		if errors.Is(err, model.ErrURLAlreadyExists) {
			return &proto.URLShortenResponse{Result: shortURL}, nil
		}
		log.Error().Err(err).Str("url", req.Url).Msg("failed to shorten url")
		return nil, status.Error(codes.Internal, "failed to shorten url")
	}

	return &proto.URLShortenResponse{Result: shortURL}, nil
}

// ExpandURL обрабатывает получение длинного URL по короткому идентификатору.
func (h *GRPCHandler) ExpandURL(ctx context.Context, req *proto.URLExpandRequest) (*proto.URLExpandResponse, error) {

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "empty id")
	}

	longURL, err := h.store.GetLongURL(ctx, req.Id)
	if err != nil {
		if errors.Is(err, model.ErrURLDeleted) {
			return nil, status.Error(codes.NotFound, "url deleted")
		}
		log.Error().Err(err).Str("id", req.Id).Msg("failed to get long url")
		return nil, status.Error(codes.NotFound, "url not found")
	}

	return &proto.URLExpandResponse{Result: longURL}, nil
}

// ListUserURLs возвращает список URL текущего пользователя.
func (h *GRPCHandler) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*proto.UserURLsResponse, error) {

	userUUID := getUserFromGRPC(ctx)
	if userUUID == "" {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid authorization")
	}

	urls, err := h.store.GetUserURLs(ctx, userUUID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userUUID).Msg("failed to get user urls")
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

// getUserFromGRPC извлекает userUUID из gRPC metadata (authorization).
func getUserFromGRPC(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return ""
	}
	return auth.GetUser(authHeaders[0])
}
