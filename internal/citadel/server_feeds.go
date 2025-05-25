package citadel

import (
	"net/http"

	"github.com/jdholdren/seymour/internal/server"
)

func (s Server) handleCreateSubscription(w http.ResponseWriter, r *http.Request) error {
	return server.WriteJSON(w, http.StatusOK, struct{}{})
}
