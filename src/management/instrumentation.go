package management

import (
	"errors"
	"fmt"
)

type InstrumentationEntry struct {
	Key         string
	Description string
}

type InstrumentationDetail struct {
	InstrumentationEntry
	Data any
}

type InstrumentationProvider interface {
	Name() string
	List() []InstrumentationEntry
	Detail(key string) InstrumentationDetail
}

var instrumentationProviders = make(map[string]InstrumentationProvider, 0)

func RegisterProvider(provider InstrumentationProvider) error {
	_, ok := instrumentationProviders[provider.Name()]
	if ok {
		return errors.New(fmt.Sprintf("Provider with name %s already registered", provider.Name()))
	}

	instrumentationProviders[provider.Name()] = provider
	RefreshInstrumentationEndpoints()

	return nil
}

func GetProviders() []InstrumentationProvider {
	list := make([]InstrumentationProvider, 0)

	for _, v := range instrumentationProviders {
		list = append(list, v)
	}

	return list
}

func GetProvider(name string) (InstrumentationProvider, bool) {
	ret, ok := instrumentationProviders[name]
	return ret, ok
}
