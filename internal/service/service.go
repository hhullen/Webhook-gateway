package service

import (
	"net"
)

type ISession interface {
	IsClosed() bool
	Open() (net.Conn, error)
	CloseChan() <-chan struct{}
}

type Service struct {
	sessions map[string]ISession
}

// func NewService() *Service {

// }
