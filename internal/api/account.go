package api

import (
	"encoding/json"
	"net/http"

	seyerrs "github.com/jdholdren/seymour/internal/errors"
	"github.com/jdholdren/seymour/internal/serverutil"

	goaway "github.com/TwiN/go-away"
)

type promptPrecheckRequest struct {
	Prompt string `json:"prompt"`
}

// This route is used to aid the front-end with validation, like running a profanity check.
func (s Server) postPromptPrecheck(w http.ResponseWriter, r *http.Request) error {
	var body promptPrecheckRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return seyerrs.E(err, http.StatusBadRequest)
	}

	// Run a length and profanity check.
	//
	// Since this is being fed to the LLM, it's imperative that we're trying to keep the input
	// rather clean.
	const maxLength = 5024
	if len(body.Prompt) > maxLength {
		return seyerrs.E("prompt too long", http.StatusUnprocessableEntity)
	}
	if goaway.IsProfane(body.Prompt) {
		return seyerrs.E("profanity detected in prompt", http.StatusUnprocessableEntity)
	}

	return serverutil.WriteJSON(w, http.StatusOK, struct{}{})
}

// This route is used to aid the front-end with validation, like running a profanity check.
func (s Server) postPrompt(w http.ResponseWriter, r *http.Request) error {
	var body promptPrecheckRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return seyerrs.E(err, http.StatusBadRequest)
	}

	// Get the session to imply the user
	state := session(r, s.secureCookie)

	// Run a length and profanity check.
	//
	// Since this is being fed to the LLM, it's imperative that we're trying to keep the input
	// rather clean.
	const maxLength = 5024
	if len(body.Prompt) > maxLength {
		return seyerrs.E("prompt too long", http.StatusUnprocessableEntity)
	}
	if goaway.IsProfane(body.Prompt) {
		return seyerrs.E("profanity detected in prompt", http.StatusUnprocessableEntity)
	}

	// Persist the prompt to the user
	if err := s.repo.UpdateUserPrompt(r.Context(), state.UserID, body.Prompt); err != nil {
		return seyerrs.E(err, http.StatusInternalServerError)
	}

	return serverutil.WriteJSON(w, http.StatusOK, struct{}{})
}
