package main

import (
	"net/http"
)

func setLocalAPIHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self';")
	w.Header().Del("Access-Control-Allow-Origin")
}

type secureResponseWriter struct {
	http.ResponseWriter
}

func (w secureResponseWriter) WriteHeader(statusCode int) {
	setLocalAPIHeaders(w.ResponseWriter)
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w secureResponseWriter) Write(data []byte) (int, error) {
	setLocalAPIHeaders(w.ResponseWriter)
	return w.ResponseWriter.Write(data)
}

func (w secureResponseWriter) Flush() {
	setLocalAPIHeaders(w.ResponseWriter)
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
