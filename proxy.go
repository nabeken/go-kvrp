package hugoreview

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
)

type ReverseProxyHandler struct {
	store Store
}

func NewHandler(s Store) *ReverseProxyHandler {
	return &ReverseProxyHandler{store: s}
}

func (h *ReverseProxyHandler) director(req *http.Request) {
	req.URL.Scheme = "http"
	c := h.store.GetHost(req.Host)
	req.URL.Host = c.Host
}

func (h *ReverseProxyHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	c := h.store.GetHost(req.Host)
	if c.Host == "" {
		http.Error(rw, "Bad Request", http.StatusBadRequest)
		return
	}
	rp := &httputil.ReverseProxy{Director: h.director}
	rp.ServeHTTP(rw, req)
}

func Getenv(key, defVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defVal
	}
	return val
}

func Addr() string {
	host := Getenv("HOST", "127.0.0.1")
	port := Getenv("PORT", "8000")
	return fmt.Sprintf("%s:%s", host, port)
}
