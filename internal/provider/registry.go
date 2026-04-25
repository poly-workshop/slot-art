package provider

import (
	"fmt"
	"sort"
)

var registry = make(map[string]Provider)

func Register(p Provider) {
	registry[p.Name()] = p
}

func Get(name string) (Provider, error) {
	p, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
	return p, nil
}

func List() []string {
	names := make([]string, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
