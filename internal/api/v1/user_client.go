package api

import (
	"net/http"
	"strings"

	websocketadapter "webhook-gateway/internal/websocket_adapter"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func (a *API) setupWebsocketHandlers() {
	a.router.Handle("/connect", http.HandlerFunc(a.connectUserClient))
}

func (a *API) connectUserClient(w http.ResponseWriter, r *http.Request) {
	authToken, hasToken := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !hasToken {
		http.Error(w, "auth token required", http.StatusUnauthorized)
		return
	}

	tunnelId := r.Header.Get("X-Tunnel-ID")
	if tunnelId == "" {
		http.Error(w, "tunnel id required", http.StatusUnauthorized)
		return
	}

	verified, err := a.auth.VerifyAuthToken(authToken)
	if err != nil {
		const msg = "failed verifying token"
		a.logger.ErrorKV(msg, "error", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	if !verified {
		http.Error(w, "invalid auth token", http.StatusUnauthorized)
		return
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		const msg = "failed upgrading to WebSocket protocol"
		a.logger.ErrorKV(msg, "error", err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	conn := websocketadapter.NewWsReadWriteCloser(wsConn)

	started := a.whService.StartSession(conn, tunnelId)
	if !started {
		http.Error(w, "failed starting client-server session", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
