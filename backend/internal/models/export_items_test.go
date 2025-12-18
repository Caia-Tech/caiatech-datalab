package models

import (
	"encoding/json"
	"testing"
)

func TestDerivePairsFromItemData_UserAssistant(t *testing.T) {
	data := json.RawMessage(`{"user":"Hi","assistant":"Hello"}`)
	pairs := derivePairsFromItemData(data, ExportOptions{Context: "none"})
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].User != "Hi" {
		t.Fatalf("unexpected user: %q", pairs[0].User)
	}
	if pairs[0].Assistant != "Hello" {
		t.Fatalf("unexpected assistant: %q", pairs[0].Assistant)
	}
}

func TestDerivePairsFromItemData_Messages(t *testing.T) {
	data := json.RawMessage(`{"messages":[{"role":"user","content":"Hi"},{"role":"assistant","content":"Hello"}]}`)
	pairs := derivePairsFromItemData(data, ExportOptions{Context: "none"})
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].User != "Hi" {
		t.Fatalf("unexpected user: %q", pairs[0].User)
	}
	if pairs[0].Assistant != "Hello" {
		t.Fatalf("unexpected assistant: %q", pairs[0].Assistant)
	}
}

func TestDerivePairsFromItemData_UnrecognizedShape(t *testing.T) {
	data := json.RawMessage(`["not","an","object"]`)
	pairs := derivePairsFromItemData(data, ExportOptions{Context: "none"})
	if len(pairs) != 0 {
		t.Fatalf("expected 0 pairs, got %d", len(pairs))
	}
}

