package go_remote

import (
	"context"
	"net/http"
)

type ContextProvider func(ctx context.Context, r *http.Request) context.Context
type ContextReaction func(ctx context.Context, w http.ResponseWriter, key interface{})

type contextStore struct {
	order []ContextProvider
	reaction []ContextReaction
}

func newContextStore() *contextStore {
	return &contextStore{order: []ContextProvider{}, reaction: []ContextReaction{}}
}

// AddProvider adds a handler which will fill context of the remove call from the request object
func (s *contextStore) AddProvider(p ContextProvider) {
	s.order = append(s.order, p)
}

// RegisterProvider adds a factory method for parameters of remote methods
func (s *contextStore) AddReaction(p ContextReaction) {
	s.reaction = append(s.reaction, p)
}

func (s *contextStore) FromRequest(r *http.Request) context.Context {
	ctx := r.Context()
	for _, prov := range s.order {
		ctx = prov(ctx, r)
	}

	return ctx
}

func (s *contextStore) ToResponse(w http.ResponseWriter, ctx context.Context, key interface{}) {
	for _, prov := range s.reaction {
		prov(ctx, w, key)
	}
}