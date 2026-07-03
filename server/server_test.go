package server

import (
	"testing"

	"github.com/feenlace/mcp-1c/onec"
)

func TestNewServer(t *testing.T) {
	client := onec.NewClient("http://localhost:8080/mcp", "", "")
	s := New("test", client, nil, nil)
	if s == nil {
		t.Fatal("expected non-nil server")
	}
}
