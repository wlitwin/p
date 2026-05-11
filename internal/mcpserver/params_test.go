package mcpserver

import (
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func makeReq(args map[string]any) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	return req
}

func TestParamsRequireAllPresent(t *testing.T) {
	req := makeReq(map[string]any{
		"project": "myproj",
		"list":    "tasks",
		"item_id": "1",
	})

	p := newParams(req)
	proj := p.require("project")
	list := p.require("list")
	itemID := p.require("item_id")

	if r := p.error(); r != nil {
		t.Fatal("expected no error when all params present")
	}
	if proj != "myproj" {
		t.Errorf("project = %q, want %q", proj, "myproj")
	}
	if list != "tasks" {
		t.Errorf("list = %q, want %q", list, "tasks")
	}
	if itemID != "1" {
		t.Errorf("item_id = %q, want %q", itemID, "1")
	}
}

func TestParamsRequireMissingSingle(t *testing.T) {
	req := makeReq(map[string]any{
		"project": "myproj",
		"list":    "",
	})

	p := newParams(req)
	p.require("project")
	p.require("list")

	r := p.error()
	if r == nil {
		t.Fatal("expected error for missing list")
	}

	// Extract error text
	text := ""
	for _, c := range r.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			text = tc.Text
		}
	}
	if !strings.Contains(text, "list") {
		t.Errorf("error should mention 'list': %s", text)
	}
}

func TestParamsRequireMissingMultiple(t *testing.T) {
	req := makeReq(map[string]any{})

	p := newParams(req)
	p.require("project")
	p.require("list")
	p.require("item_id")

	r := p.error()
	if r == nil {
		t.Fatal("expected error for all missing params")
	}

	text := ""
	for _, c := range r.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			text = tc.Text
		}
	}
	if !strings.Contains(text, "project") || !strings.Contains(text, "list") || !strings.Contains(text, "item_id") {
		t.Errorf("error should list all missing params: %s", text)
	}
}

func TestParamsOptionalWithDefault(t *testing.T) {
	req := makeReq(map[string]any{})

	p := newParams(req)
	val := p.optional("priority", "now")
	if val != "now" {
		t.Errorf("optional = %q, want %q", val, "now")
	}
}

func TestParamsOptionalWithValue(t *testing.T) {
	req := makeReq(map[string]any{
		"priority": "backlog",
	})

	p := newParams(req)
	val := p.optional("priority", "now")
	if val != "backlog" {
		t.Errorf("optional = %q, want %q", val, "backlog")
	}
}

func TestParamsOptionalBool(t *testing.T) {
	req := makeReq(map[string]any{
		"clear": true,
	})

	p := newParams(req)
	if !p.optionalBool("clear", false) {
		t.Error("optionalBool should return true")
	}

	// Missing bool should return default
	if p.optionalBool("nonexistent", false) {
		t.Error("optionalBool should return false for missing key")
	}
}

func TestParamsNoErrorWhenNoRequireCalled(t *testing.T) {
	req := makeReq(map[string]any{})
	p := newParams(req)
	if r := p.error(); r != nil {
		t.Fatal("expected no error when no require calls made")
	}
}
