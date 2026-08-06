package util

import "strings"

func DiffEndpoint(s, o []string) (v []string) {
f:
	for _, i := range s {
		for _, j := range o {
			if endpointCmp(i, j) {
				continue f
			}
		}

		v = append(v, i)
	}

	return
}

func endpointCmp(s, o string) bool {
	return endpointCmpMethod(s, o) || endpointCmpURL(s, o)
}

func endpointCmpMethod(s, o string) bool {
	s = strings.ReplaceAll(s, "_", "")
	o = strings.ReplaceAll(o, "_", "")
	return strings.EqualFold(s, o)
}

func endpointCmpURL(s, o string) bool {
	return httpEndpointEqual(httpEndpointUnify(strings.Split(s, "/")), httpEndpointUnify(strings.Split(o, "/")))
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
