package apihandlers

import (
	"net/http"

	"github.com/DarmawanEfendi/scheduler/internal/api-handlers/httputil"
)

type HomeHandler struct{}

func NewHomeHandler() http.Handler {
	handler := &HomeHandler{}
	return handler
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	httputil.ReturnSuccess(w, http.StatusOK, "Welcome Home!", nil)
}
