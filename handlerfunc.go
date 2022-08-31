package ezy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type Void struct{}

type HandlerFunc[T any] func(ctx Context, p *T) error

func (h HandlerFunc[T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	input := new(T)

	if r.ContentLength != 0 {
		err := json.NewDecoder(r.Body).Decode(input)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(NewError(http.StatusBadRequest, err.Error()))
			return
		}

		if v, ok := any(input).(Validator); ok {
			if err = v.Validate(r.Context()); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(NewError(http.StatusBadRequest, err.Error()))
				return
			}
		}
	}

	c := &ctx{w: w, r: r, b: &bytes.Buffer{}, c: 200}
	if err := h(c, input); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewError(http.StatusInternalServerError, err.Error()))
		return
	}

	w.WriteHeader(c.c)
	if _, err := io.Copy(w, c.b); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewError(http.StatusInternalServerError, err.Error()))
		return
	}
}