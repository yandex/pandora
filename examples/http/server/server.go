package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/yandex/pandora/lib/str"
)

const (
	defaultPort = "8091"

	userCount         = 10
	userMultiplicator = 1000
	itemMultiplicator = 100
)

type StatisticBodyResponse struct {
	Code200 map[int64]uint64 `json:"200"`
	Code400 uint64           `json:"400"`
	Code500 uint64           `json:"500"`
}

type StatisticResponse struct {
	Auth StatisticBodyResponse `json:"auth"`
	List StatisticBodyResponse `json:"list"`
	Item StatisticBodyResponse `json:"item"`
}

func checkContentTypeAndMethod(r *http.Request, methods []string) (int, error) {
	contentType := r.Header.Get("Content-Type")
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return http.StatusBadRequest, errors.New("malformed Content-Type header")
	}

	if mt != "application/json" {
		return http.StatusUnsupportedMediaType, errors.New("header Content-Type must be application/json")
	}

	for _, method := range methods {
		if r.Method == method {
			return 0, nil
		}
	}
	return http.StatusMethodNotAllowed, errors.New("method not allowed")
}

func (s *Server) checkAuthorization(r *http.Request) (int64, int, error) {
	authHeader := r.Header.Get("Authorization")
	authHeader = strings.Replace(authHeader, "Bearer ", "", 1)
	s.mu.RLock()
	userID := s.keys[authHeader]
	s.mu.RUnlock()

	if userID == 0 {
		return 0, http.StatusUnauthorized, errors.New("StatusUnauthorized")
	}
	return userID, 0, nil
}

func (s *Server) authHandler(w http.ResponseWriter, r *http.Request) {
	code, err := checkContentTypeAndMethod(r, []string{http.MethodPost})
	if err != nil {
		if code >= 500 {
			s.stats.IncAuth500()
		} else {
			s.stats.IncAuth400()
		}
		http.Error(w, err.Error(), code)
		return
	}

	user := struct {
		UserID int64 `json:"user_id"`
	}{}
	err = json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		s.stats.IncAuth500()
		http.Error(w, "Incorrect body", http.StatusNotAcceptable)
		return
	}
	if user.UserID > userCount {
		s.stats.IncAuth400()
		http.Error(w, "Incorrect user_id", http.StatusBadRequest)
		return
	}

	s.stats.IncAuth200(user.UserID)

	var authKey string
	s.mu.RLock()
	for k, v := range s.keys {
		if v == user.UserID {
			authKey = k
			break
		}
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(fmt.Sprintf(`{"auth_key": "%s"}`, authKey)))
}

func (s *Server) listHandler(w http.ResponseWriter, r *http.Request) {
	code, err := checkContentTypeAndMethod(r, []string{http.MethodGet})
	if err != nil {
		if code >= 500 {
			s.stats.IncList500()
		} else {
			s.stats.IncList400()
		}
		http.Error(w, err.Error(), code)
		return
	}

	userID, code, err := s.checkAuthorization(r)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	s.stats.IncList200(userID)

	// Logic
	userID *= userMultiplicator
	result := make([]string, itemMultiplicator)
	for i := int64(0); i < itemMultiplicator; i++ {
		result[i] = strconv.FormatInt(userID+i, 10)
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(fmt.Sprintf(`{"items": [%s]}`, strings.Join(result, ","))))
}

func (s *Server) orderHandler(w http.ResponseWriter, r *http.Request) {
	code, err := checkContentTypeAndMethod(r, []string{http.MethodPost})
	if err != nil {
		if code >= 500 {
			s.stats.IncOrder500()
		} else {
			s.stats.IncOrder400()
		}
		http.Error(w, err.Error(), code)
		return
	}

	userID, code, err := s.checkAuthorization(r)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	// Logic
	itm := struct {
		ItemID int64 `json:"item_id"`
	}{}
	err = json.NewDecoder(r.Body).Decode(&itm)
	if err != nil {
		s.stats.IncOrder500()
		http.Error(w, "Incorrect body", http.StatusNotAcceptable)
		return
	}

	ranger := userID * userMultiplicator
	if itm.ItemID < ranger || itm.ItemID >= ranger+itemMultiplicator {
		s.stats.IncOrder400()
		http.Error(w, "Incorrect user_id", http.StatusBadRequest)
		return
	}

	s.stats.IncOrder200(userID)

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(fmt.Sprintf(`{"item": %d}`, itm.ItemID)))
}

func (s *Server) resetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.stats.Reset()

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status": "ok"}`))
}

func (s *Server) statisticHandler(w http.ResponseWriter, r *http.Request) {
	response := StatisticResponse{
		Auth: StatisticBodyResponse{
			Code200: s.stats.Auth200,
			Code400: s.stats.auth400.Load(),
			Code500: s.stats.auth500.Load(),
		},
		List: StatisticBodyResponse{
			Code200: s.stats.List200,
			Code400: s.stats.list400.Load(),
			Code500: s.stats.list500.Load(),
		},
		Item: StatisticBodyResponse{
			Code200: s.stats.Order200,
			Code400: s.stats.order400.Load(),
			Code500: s.stats.order500.Load(),
		},
	}
	b, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func NewServer(addr string, log *slog.Logger, seed int64) *Server {
	keys := make(map[string]int64, userCount)
	for i := int64(1); i <= userCount; i++ {
		keys[str.RandStringRunes(64, "")] = i
	}

	result := &Server{Log: log, stats: newStats(userCount), keys: keys}
	mux := http.NewServeMux()

	mux.Handle("/auth", http.HandlerFunc(result.authHandler))
	mux.Handle("/list", http.HandlerFunc(result.listHandler))
	mux.Handle("/order", http.HandlerFunc(result.orderHandler))
	mux.Handle("/stats", http.HandlerFunc(result.statisticHandler))
	mux.Handle("/reset", http.HandlerFunc(result.resetHandler))

	ctx := context.Background()
	result.srv = &http.Server{
		Addr:    addr,
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}
	log.Info("New server created", slog.String("addr", addr), slog.Any("keys", keys))

	return result
}

type Server struct {
	srv *http.Server

	Log   *slog.Logger
	stats *Stats
	keys  map[string]int64
	mu    sync.RWMutex

	runErr chan error
	finish bool
}

func (s *Server) Err() <-chan error {
	return s.runErr
}

func (s *Server) ServeAsync() {
	go func() {
		err := s.srv.ListenAndServe()
		if err != nil {
			s.runErr <- err
		} else {
			s.runErr <- nil
		}
		s.finish = true
	}()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) Stats() *Stats {
	return s.stats
}
