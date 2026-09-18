// Package ui contains the generic server-rendered admin components.
//
//go:generate templ generate
package ui

import (
	"context"
	"net/http"

	"github.com/malero/ent-admin/internal/view"
)

// Render renders a page using the generic server-rendered Templ components.
// The runtime owns status selection; the UI owns HTML structure and escaping.
func Render(ctx context.Context, w http.ResponseWriter, page view.Page) error {
	writeHTMLStatus(w, page.StatusCode)

	return pageComponent(page).Render(ctx, w)
}

// RenderDetailFragment renders the VSN response for a detail-preview request.
// It intentionally omits the document shell so VSN can swap it into the list
// page while the same URL still renders a complete page for ordinary clients.
func RenderDetailFragment(ctx context.Context, w http.ResponseWriter, page view.Page) error {
	writeHTMLStatus(w, page.StatusCode)
	return detailFragment(page).Render(ctx, w)
}

func writeHTMLStatus(w http.ResponseWriter, status int) {
	if status == 0 {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
}
