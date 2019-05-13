package remote

import (
	"encoding/json"
)

type dataProvider struct {
	*serviceProvider
}

func newDataProvider(di *provider) *dataProvider {
	t := dataProvider{}
	t.serviceProvider = newServiceProvider(di)
	return &t
}

// MarshalJSON converts object to JSON struct
func (s *dataProvider) ToJSON(state callState) ([]byte, error) {
	all := make(map[string][]byte)
	for key := range s.methods {
		val, err := s.callAndResolve(s.methods[key], state)
		if err != nil {
			return nil, err
		}

		data, _ := json.Marshal(val)
		all[key] = data
	}

	for key := range s.services {
		all[key], _ = s.services[key].ToJSON()
	}

	return json.Marshal(all)
}
