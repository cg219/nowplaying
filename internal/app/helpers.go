package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof"
)

func return500(w http.ResponseWriter) {
    encode(w, 500, ResponseError{ Success: false, Messaage: INTERNAL_ERROR, Code: INTERNAL_SERVER_ERROR })
}

func encode[T any](w http.ResponseWriter, status int, v T) error {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(v); err != nil {
        return fmt.Errorf("encoding: %w", err)
    }

    return nil
}

func decode[T any](r *http.Request) (T, error) {
    var v T
    if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
        return v, fmt.Errorf("decoding: %w", err)
    }

    return v, nil
}
