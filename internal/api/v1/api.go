package api

import (
	"context"
	"io"
	"net/http"
	"time"
)

var (
	readTimeout  = time.Second * 10
	writeTimeout = time.Second * 10
)

type IServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

type IRouter interface {
	Handle(pattern string, handler http.Handler)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

type ILogger interface {
	InfoKV(message string, argsKV ...any)
	// WarnKV(message string, argsKV ...any)
	ErrorKV(message string, argsKV ...any)
	// FatalKV(message string, argsKV ...any)
}

type ILimiter interface {
	Allow(key string, perSecond int) (bool, error)
}

type IWebhookGatewayService interface {
	StartSession(conn io.ReadWriteCloser, authToken, tunnelId string) error
	OpenStream(tunnelId string) (io.ReadWriteCloser, error)
}

type ResponseWriterInterceptor struct {
	rw   http.ResponseWriter
	code int
}

func (rwi *ResponseWriterInterceptor) Header() http.Header {
	return rwi.rw.Header()
}

func (rwi *ResponseWriterInterceptor) Write(data []byte) (int, error) {
	return rwi.rw.Write(data)
}

func (rwi *ResponseWriterInterceptor) WriteHeader(statusCode int) {
	rwi.code = statusCode
	rwi.rw.WriteHeader(statusCode)
}

type API struct {
	logger  ILogger
	limiter ILimiter
	server  IServer
	router  IRouter
	service IWebhookGatewayService
}

func NewAPI(
	address string,
	lim ILimiter,
	log ILogger) (*API, error) {
	router := http.NewServeMux()

	server := &http.Server{
		Addr:         address,
		Handler:      mainMiddleware(router, log),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	return buildAPI(log, lim, server, router), nil
}

func buildAPI(
	log ILogger,
	lim ILimiter,
	srv IServer,
	rou IRouter) *API {
	api := &API{
		logger:  log,
		limiter: lim,
		server:  srv,
		router:  rou,
	}

	api.setupWebsocketHandlers()
	api.setupWebhookHandlers()

	return api
}

func (a *API) StartListening() error {
	a.logger.InfoKV("Start listening")
	return a.server.ListenAndServe()
}

func (a *API) Stop(ctx context.Context) error {
	a.logger.InfoKV("Server Shutdown")
	return a.server.Shutdown(ctx)
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

func mainMiddleware(next http.Handler, log ILogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts := time.Now()

		rwi := &ResponseWriterInterceptor{rw: w}
		next.ServeHTTP(rwi, r)

		te := time.Since(ts)
		log.InfoKV("Request", "method", r.Method, "url", r.URL.String(), "duration(ms)", te.Milliseconds(), "status_code", rwi.code)
	})
}
