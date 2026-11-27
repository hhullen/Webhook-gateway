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
	FatalKV(message string, argsKV ...any)
}

type ILimiter interface {
	Allow(key string, perSecond int) (bool, error)
}

type IWebhookGatewayService interface {
	StartSession(conn io.ReadWriteCloser, tunnelId string) bool
	OpenStream(tunnelId string) (io.ReadWriteCloser, bool)
	Stop(ctx context.Context)
}

type IAuthService interface {
	VerifyAuthToken(authTocken string) (bool, error)
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
	logger    ILogger
	limiter   ILimiter
	server    IServer
	router    IRouter
	whService IWebhookGatewayService
	auth      IAuthService
}

func NewAPI(
	address string,
	wh IWebhookGatewayService,
	auth IAuthService,
	lim ILimiter,
	log ILogger) (*API, error) {
	router := http.NewServeMux()

	server := &http.Server{
		Addr:         address,
		Handler:      mainMiddleware(router, log),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	return buildAPI(wh, auth, log, lim, server, router), nil
}

func buildAPI(
	wh IWebhookGatewayService,
	auth IAuthService,
	log ILogger,
	lim ILimiter,
	srv IServer,
	rou IRouter) *API {
	api := &API{
		whService: wh,
		auth:      auth,
		logger:    log,
		limiter:   lim,
		server:    srv,
		router:    rou,
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
	a.whService.Stop(ctx)
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
