package handlers

import (
	"net/http"

	"github.com/bnb-chain/blob-syncer/service"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
	header     http.Header
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK, []byte{}, http.Header{}}
}

func (rw *responseWriter) Write(body []byte) (int, error) {
	rw.body = body
	return rw.ResponseWriter.Write(body)
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.header = rw.ResponseWriter.Header()
	rw.ResponseWriter.WriteHeader(code)
}

func Error(err error) (int64, string) {
	switch e := err.(type) {
	case service.Err:
		return e.Code, e.Message
	case nil:
		return service.NoErr.Code, service.NoErr.Message
	default:
		return service.InternalErr.Code, err.Error()
	}
}
