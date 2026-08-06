package internal

import (
	"bytes"
	"os"
)

// KeepAutoGen is the main entry to decide whether a Go file should be kept
// because it is auto generated. It delegates to specific detectors (ANTLR, etc.)
// so adding more frameworks is straightforward.
func KeepAutoGen(filename string) bool {
	raw, err := os.ReadFile(filename)
	if err != nil {
		return false
	}
	head := header(raw)

	// add more checks here (e.g. protoc, swagger)
	if isANTLR(head) {
		return true
	}
	return false
}

// header trims content to a small prefix; generation markers live near the top.
func header(raw []byte) []byte {
	if len(raw) > 4096 {
		return raw[:4096]
	}
	return raw
}

// isANTLR detects ANTLR-generated Go files.
// Typical header: // Code generated from foo.g4 by ANTLR 4.x. DO NOT EDIT.
func isANTLR(raw []byte) bool {
	lower := bytes.ToLower(raw)
	if bytes.Contains(lower, []byte("by antlr")) {
		return true
	}
	if bytes.Contains(lower, []byte("antlr 4.")) && bytes.Contains(lower, []byte("do not edit")) {
		return true
	}
	return false
}
