package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	seyerrs "github.com/jdholdren/seymour/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostPromptPrecheck_ContainsProfanity(t *testing.T) {
	var (
		req = httptest.NewRequest(http.MethodPost, "/account/prompt:precheck", strings.NewReader(`{"prompt": "f u c k it"}`))
		rec = httptest.NewRecorder()
		s   = newTestApiServer(t)
	)

	err := s.postPromptPrecheck(rec, req)
	require.Error(t, err)

	var seyerr *seyerrs.Error
	require.ErrorAs(t, err, &seyerr)
	assert.Equal(t, seyerr.Status, http.StatusUnprocessableEntity)
}
