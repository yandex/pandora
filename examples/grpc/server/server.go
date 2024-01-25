package server

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	userCount         = 10
	userMultiplicator = 1000
	itemMultiplicator = 100
)

func NewServer(logger *slog.Logger, seed int64) *GRPCServer {
	r := rand.New(rand.NewSource(seed))
	var randStringRunes = func(n int) string {
		var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
		b := make([]rune, n)
		for i := range b {
			b[i] = letterRunes[r.Intn(len(letterRunes))]
		}
		return string(b)
	}

	keys := make(map[string]int64, userCount)
	for i := int64(1); i <= userCount; i++ {
		keys[randStringRunes(64)] = i
	}
	logger.Info("New server created", slog.Any("keys", keys))

	return &GRPCServer{Log: logger, keys: keys, stats: newStats(userCount)}
}

type GRPCServer struct {
	UnimplementedTargetServiceServer
	Log   *slog.Logger
	stats *Stats
	keys  map[string]int64
	mu    sync.RWMutex
}

var _ TargetServiceServer = (*GRPCServer)(nil)

func (s *GRPCServer) Auth(ctx context.Context, request *AuthRequest) (*AuthResponse, error) {
	userID, token, err := s.checkLoginPass(request.GetLogin(), request.GetPass())
	if err != nil {
		s.stats.IncAuth400()
		return nil, status.Error(codes.InvalidArgument, "invalid credentials")
	}
	result := &AuthResponse{
		UserId: userID,
		Token:  token,
	}
	s.stats.IncAuth200(userID)
	return result, nil
}

func (s *GRPCServer) List(ctx context.Context, request *ListRequest) (*ListResponse, error) {
	s.mu.RLock()
	userID := s.keys[request.Token]
	s.mu.RUnlock()
	if userID == 0 {
		s.stats.IncList400()
		return nil, status.Error(codes.InvalidArgument, "invalid token")
	}
	if userID != request.UserId {
		s.stats.IncList400()
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	// Logic
	result := &ListResponse{}
	userID *= userMultiplicator
	result.Result = make([]*ListItem, itemMultiplicator)
	for i := int64(0); i < itemMultiplicator; i++ {
		result.Result[i] = &ListItem{ItemId: userID + i}
	}
	s.stats.IncList200(request.UserId)
	return result, nil
}

func (s *GRPCServer) Order(ctx context.Context, request *OrderRequest) (*OrderResponse, error) {
	s.mu.RLock()
	userID := s.keys[request.Token]
	s.mu.RUnlock()
	if userID == 0 {
		s.stats.IncOrder400()
		return nil, status.Error(codes.InvalidArgument, "invalid token")
	}
	if userID != request.UserId {
		s.stats.IncOrder400()
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	// Logic
	ranger := userID * userMultiplicator
	if request.ItemId < ranger || request.ItemId >= ranger+itemMultiplicator {
		s.stats.IncOrder400()
		return nil, status.Error(codes.InvalidArgument, "invalid item_id")
	}

	result := &OrderResponse{}
	result.OrderId = request.ItemId + 12345
	s.stats.IncOrder200(userID)
	return result, nil
}

func (s *GRPCServer) Stats(ctx context.Context, _ *StatsRequest) (*StatsResponse, error) {
	result := &StatsResponse{
		Auth: &StatisticBodyResponse{
			Code200: s.stats.auth200,
			Code400: s.stats.auth400.Load(),
			Code500: s.stats.auth500.Load(),
		},
		List: &StatisticBodyResponse{
			Code200: s.stats.list200,
			Code400: s.stats.list400.Load(),
			Code500: s.stats.list500.Load(),
		},
		Order: &StatisticBodyResponse{
			Code200: s.stats.order200,
			Code400: s.stats.order400.Load(),
			Code500: s.stats.order500.Load(),
		},
	}
	return result, nil
}

func (s *GRPCServer) Reset(ctx context.Context, request *ResetRequest) (*ResetResponse, error) {
	s.stats.Reset()

	result := &ResetResponse{
		Auth: &StatisticBodyResponse{
			Code200: s.stats.auth200,
			Code400: s.stats.auth400.Load(),
			Code500: s.stats.auth500.Load(),
		},
		List: &StatisticBodyResponse{
			Code200: s.stats.list200,
			Code400: s.stats.list400.Load(),
			Code500: s.stats.list500.Load(),
		},
		Order: &StatisticBodyResponse{
			Code200: s.stats.order200,
			Code400: s.stats.order400.Load(),
			Code500: s.stats.order500.Load(),
		},
	}
	return result, nil
}

func (s *GRPCServer) checkLoginPass(login string, pass string) (int64, string, error) {
	userID, err := strconv.ParseInt(login, 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid login %s", login)
	}
	if login != pass {
		return 0, "", fmt.Errorf("invalid login %s or pass %s", login, pass)
	}
	token := ""
	s.mu.RLock()
	for k, v := range s.keys {
		if v == userID {
			token = k
			break
		}
	}
	s.mu.RUnlock()
	if token == "" {
		return 0, "", fmt.Errorf("invalid login %s and pass %s", login, pass)
	}

	return userID, token, nil
}
