package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"arupa/internal/auth"
	"arupa/internal/conf"
	"arupa/internal/netx"
)

type optionalField[T any] struct {
	Present bool
	Null    bool
	Value   T
}

func (field *optionalField[T]) UnmarshalJSON(data []byte) error {
	field.Present = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		field.Null = true
		return nil
	}
	return json.Unmarshal(data, &field.Value)
}

func management(capability conf.APICapability, next http.HandlerFunc) http.HandlerFunc {
	return auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		if !conf.APIEnabled(capability) {
			_ = netx.WriteNotFound(w)
			return
		}
		next(w, r)
	})
}

func decodeRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		_ = netx.WriteBadRequest(w, "Invalid request body")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		_ = netx.WriteBadRequest(w, "Request body must contain one JSON value")
		return false
	}
	return true
}
