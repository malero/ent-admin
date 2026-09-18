package ui

import (
	_ "embed"
	"net/http"
)

//go:embed assets/vsn.min.js
var vsnScript []byte

// ServeVSNScript serves the pinned VSN runtime used by the progressive UI.
func ServeVSNScript(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, err := w.Write(vsnScript)
	return err
}
