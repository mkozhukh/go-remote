package go_remote

import (
	"context"
	"reflect"
)

type ServiceAPI map[string]int
type API struct {
	Services map[string]ServiceAPI `json:"api"`
	Data     map[string]interface{}
	Key      string
}

// JSON returns a json string representation of the end point
func (s *Server) GetAPI(ctx context.Context) API {
	out := API{}
	out.Services = make(map[string]ServiceAPI)
	out.Data = make(map[string]interface{})

	token := ctx.Value(TokenValue)
	strKey, ok := token.(string)
	if ok {
		out.Key = strKey
	}

	for key, value := range s.services {
		out.Services[key] = value.GetAPI()
	}

	for key, value := range s.data {
		if value.isConstant {
			out.Data[key] = value.value
		} else {
			raw, err := s.Dependencies.Value(value.rtype, ctx)
			if err != nil {
				log.Errorf("can't resolve api variable: %s\n%f", key, err)
			} else {
				if raw.Kind() == reflect.Ptr {
					raw = raw.Elem()
				}
				out.Data[key] = raw.Interface()
			}
		}
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
