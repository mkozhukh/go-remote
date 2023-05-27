package go_remote

import (
	"context"
	"reflect"
)

type ServiceAPI map[string]int
type API struct {
	Services  map[string]ServiceAPI  `json:"api"`
	Data      map[string]interface{} `json:"data"`
	WebSocket bool                   `json:"websocket,omitempty"`
}

// JSON returns a json string representation of the end point
func (s *Server) GetAPI(ctx context.Context) API {
	out := API{}
	out.Services = make(map[string]ServiceAPI)
	out.Data = make(map[string]interface{})

	for key, value := range s.services {
		out.Services[key] = value.GetAPI()
	}

	for key, value := range s.data {
		if value.isConstant {
			out.Data[key] = value.value
		} else {
			raw, ok, err := s.Dependencies.Value(value.rtype, ctx)
			if !ok {
				log.Error("can't resolve api variable", "var", key)
				continue
			}

			if err != nil {
				log.Error("error during resolving api variable", "var", key, "err", err)
				continue
			}

			if raw.Kind() == reflect.Ptr {
				raw = raw.Elem()
			}
			out.Data[key] = raw.Interface()
		}
	}

	if s.config.WebSocket {
		out.WebSocket = true
	}

	return out
}

func (s *service) GetAPI() ServiceAPI {
	out := ServiceAPI(make(map[string]int))

	for key := range s.method {
		out[key] = 1
	}

	return out
}
