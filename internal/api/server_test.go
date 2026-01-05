package api

import "testing"

func newTestApiServer(t *testing.T) *Server {
	t.Helper()
	// TODO: Eventually set up a new server for the test with in-mem sqlite and a mock for temporal.
	return NewServer(ServerConfig{}, nil, nil)
}
