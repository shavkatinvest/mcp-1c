package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/feenlace/mcp-1c/onec"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestIsWriteBlacklisted(t *testing.T) {
	cases := []struct {
		typeName string
		want     bool
	}{
		{"dibank_ПлатежноеПоручениеИсходящее", true},
		{"DIBANK_ЗаявкаНаОткрытиеСчетов", true},
		{"дибанк_Выписка", true},
		{"РеализацияТоваровУслуг", false},
		{"Контрагенты", false},
		{"", false},
	}
	for _, c := range cases {
		got := IsWriteBlacklisted(c.typeName, DefaultWriteTypeBlacklist)
		if got != c.want {
			t.Errorf("IsWriteBlacklisted(%q) = %v, want %v", c.typeName, got, c.want)
		}
	}
}

func TestIsWriteBlacklisted_EmptyBlacklistDisables(t *testing.T) {
	if IsWriteBlacklisted("dibank_ПлатежноеПоручениеИсходящее", nil) {
		t.Error("expected nil blacklist to match nothing")
	}
	if IsWriteBlacklisted("dibank_ПлатежноеПоручениеИсходящее", []string{}) {
		t.Error("expected empty blacklist to match nothing")
	}
}

func TestIsWriteBlacklisted_CustomPattern(t *testing.T) {
	custom := []string{"test_*"}
	if !IsWriteBlacklisted("test_Something", custom) {
		t.Error("expected custom pattern to match")
	}
	if IsWriteBlacklisted("dibank_Anything", custom) {
		t.Error("default dibank pattern should not apply when a custom blacklist is given")
	}
}

func TestCreateDocumentHandler_Blacklisted(t *testing.T) {
	httpCalled := false
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpCalled = true
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewCreateDocumentHandler(client, DefaultWriteTypeBlacklist)

	args, _ := json.Marshal(map[string]any{
		"document_type": "dibank_ПлатежноеПоручениеИсходящее",
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "create_document", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("expected a tool-level error result, not a Go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for blacklisted document type")
	}
	if httpCalled {
		t.Error("HTTP call should not have been made for a blacklisted type")
	}
}

func TestSetDeletionMarkHandler_Blacklisted(t *testing.T) {
	httpCalled := false
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpCalled = true
	}))
	defer mockServer.Close()

	client := onec.NewClient(mockServer.URL, "", "")
	handler := NewSetDeletionMarkHandler(client, DefaultWriteTypeBlacklist)

	args, _ := json.Marshal(map[string]any{
		"object_kind": "Document",
		"type":        "dibank_ЗаявкаНаОткрытиеСчетов",
		"ref":         "11111111-1111-1111-1111-111111111111",
	})
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "set_deletion_mark", Arguments: args}}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("expected a tool-level error result, not a Go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError=true for blacklisted type")
	}
	if httpCalled {
		t.Error("HTTP call should not have been made for a blacklisted type")
	}
}
