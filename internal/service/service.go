package service

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/hashicorp/yamux"
)

type ISession interface {
	IsClosed() bool
	Open() (net.Conn, error)
	CloseChan() <-chan struct{}
	Close() error
}

type ILogger interface {
	InfoKV(message string, argsKV ...any)
	WarnKV(message string, argsKV ...any)
	ErrorKV(message string, argsKV ...any)
	FatalKV(message string, argsKV ...any)
}

type Service struct {
	mtx      sync.RWMutex
	sessions map[string]ISession
	logger   ILogger
}

func NewService(l ILogger) *Service {
	return &Service{
		sessions: make(map[string]ISession),
		logger:   l,
	}
}

func (s *Service) StartSession(conn io.ReadWriteCloser, tunnelId string) bool {
	session, err := yamux.Server(conn, yamux.DefaultConfig())
	if err != nil {
		s.logger.ErrorKV("failed creating new session", "error", err)
		return false
	}

	s.mtx.Lock()
	s.sessions[tunnelId] = session
	s.mtx.Unlock()

	return true
}

func (s *Service) OpenStream(tunnelId string) (io.ReadWriteCloser, bool) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	session, ok := s.sessions[tunnelId]
	if !ok {
		s.logger.WarnKV("no session started for tunnel", "tunnel_id", tunnelId)
		return nil, false
	}

	stream, err := session.Open()
	if err != nil {
		s.logger.WarnKV("failed open stream for tunnel", "tunnel_id", tunnelId, "error", err)
		return nil, false
	}

	return stream, true
}

func (s *Service) Stop(ctx context.Context) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for k, _ := range s.sessions {
		if err := s.sessions[k].Close(); err != nil {
			s.logger.ErrorKV("failed closing session", "error", err, "session", k)
		}
	}
}
