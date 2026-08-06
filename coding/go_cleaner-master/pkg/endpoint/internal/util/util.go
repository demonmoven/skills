package util

import (
	"strings"
)

func Any[T any](t []T, c func(T) bool) bool {
	for _, i := range t {
		if c(i) {
			return true
		}
	}
	return false
}

func AnyKey[T comparable, V any](t map[T]V, c func(T) bool) bool {
	for i := range t {
		if c(i) {
			return true
		}
	}
	return false
}

func EndpointContain(ep string, eps []string) bool {
	ep = strings.ReplaceAll(ep, "_", "")
	for _, i := range eps {
		if strings.EqualFold(ep, strings.ReplaceAll(i, "_", "")) {
			return true
		}
	}
	return false
}

var registerFunc = map[string]bool{
	"post":    true,
	"get":     true,
	"delete":  true,
	"patch":   true,
	"put":     true,
	"options": true,
	"head":    true,
	"any":     true,
}

func IsHTTPRegister(s string) bool {
	return registerFunc[strings.ToLower(s)]
}

func HTTPEndpointContains(group, path string, eps []string) bool {
	g := strings.Split(group, "/")
	p := strings.Split(path, "/")
	u := httpEndpointUnify(append(g, p...))
	for _, ep := range eps {
		if httpEndpointEqual(u, httpEndpointUnify(strings.Split(ep, "/"))) {
			return true
		}
	}
	return false
}

func httpEndpointEqual(s, o []string) bool {
	if len(s) != len(o) {
		return false
	}
	for i := range s {
		if s[i] == "*" || o[i] == "*" {
			continue
		}
		if s[i] != o[i] {
			return false
		}
	}

	return true
}

func httpEndpointUnify(s []string) []string {
	idx := 0
	for i := range s {
		if s[i] == "" {
			continue
		}
		if s[i][0] == ':' {
			s[idx] = "*"
			idx += 1
			continue
		}
		s[idx] = s[i]
		idx += 1
	}
	return s[:idx]
}
