package api

import (
	"bufio"
	"net"
	"net/http"
	"os"
	"strings"
)

func (a *API) setupWebhookHandlers() {
	a.router.Handle("/", http.HandlerFunc(a.serveWebhook))
}

func (a *API) serveWebhook(w http.ResponseWriter, r *http.Request) {
	tunnelId, extracted := a.extractTunnelIdFromHost(r.Host)
	if !extracted {
		http.Error(w, "tunnel id is incorrect or not provided", http.StatusInternalServerError)
		return
	}

	stream, opened := a.whService.OpenStream(tunnelId)
	if !opened {
		http.Error(w, "failed opening client-server stream", http.StatusInternalServerError)
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

func (a *API) extractTunnelIdFromHost(host string) (string, bool) {
	host, _, err := net.SplitHostPort(host)
	if err != nil {
		a.logger.ErrorKV("failed splitting host:port", "error", err)
		return "", false
	}

	host = strings.ToLower(strings.TrimSpace(host))

	baseDomain := os.Getenv("BASE_DOMAIN")
	if baseDomain == "" {
		a.logger.FatalKV("base bomain not provided",
			"error", "empty base domain", "requirement", "set BASE_DOMAIN environment variable")
		return "", false
	}

	suffix := baseDomain
	if baseDomain[0] != '.' {
		suffix = "." + baseDomain
	}

	tunnelId, found := strings.CutSuffix(host, suffix)
	if !found || tunnelId == "" {
		a.logger.ErrorKV("not found tunnel id")
		return "", false
	}

	if strings.Contains(tunnelId, ".") {
		return "", false
	}

	return tunnelId, true
}
