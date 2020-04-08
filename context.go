package go_remote

import (
	"context"
	"net/http"
)

type ContextProvider func(ctx context.Context, r *http.Request) context.Context

type contextStore struct {
	order []ContextProvider
}

func newContextStore() *contextStore {
	return &contextStore{order: []ContextProvider{}}
}

// RegisterProvider adds a factory method for parameters of remote methods
func (s *contextStore) AddProvider(p ContextProvider) {
	s.order = append(s.order, p)
}

func (s *contextStore) FromRequest(r *http.Request) context.Context {
	ctx := r.Context()
	for _, prov := range s.order {
		ctx = prov(ctx, r)
	}

	return ctx
}
