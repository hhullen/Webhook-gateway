package api

import (
	"bufio"
	"fmt"
	"net/http"
	ds "webhook-gateway/internal/datastruct"
)

func (a *API) setupWebhookHandlers() {
	a.router.Handle("/wh/{tunnel_id}/{rest...}", http.HandlerFunc(a.serveWebhook))
}

func (a *API) serveWebhook(w http.ResponseWriter, r *http.Request) {
	tunnelId := r.PathValue("tunnel_id")
	if tunnelId == "" {
		http.Error(w, "tunnel id required", http.StatusInternalServerError)
		return
	}

	cleanPath := fmt.Sprintf("/%s", r.PathValue("rest"))
	r.URL.Path = cleanPath
	r.URL.RawPath = ""

	stream, err := a.service.OpenStream(tunnelId)
	if err != nil {
		if err == ds.ErrIncorrectTunnelId {
			http.Error(w, "incorrect tunnel id", http.StatusBadRequest)
			return
		}

		const msg = "failed opening client-server stream"
		a.logger.ErrorKV(msg, "error", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	if err := r.Write(stream); err != nil {
		const msg = "failed delievering request"
		a.logger.ErrorKV(msg, "error", err)
		http.Error(w, msg, http.StatusBadGateway)
		return
	}

	resp, err := http.ReadResponse(bufio.NewReader(stream), r)
	if err != nil {
		const msg = "failed getting response"
		a.logger.ErrorKV(msg, "error", err)
		http.Error(w, msg, http.StatusBadGateway)
		return
	}

	if err := resp.Write(w); err != nil {
		a.logger.ErrorKV("failed writing response", "error", err)
	}
}
