package mcpserver

import (
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// params provides a fluent API for extracting and validating MCP request parameters.
// It accumulates required string parameters and returns an error result if any are missing.
//
// Usage:
//
//	p := newParams(req)
//	proj := p.require("project")
//	list := p.require("list")
//	itemID := p.require("item_id")
//	if r := p.error(); r != nil {
//	    return r, nil
//	}
type params struct {
	req     mcp.CallToolRequest
	missing []string
}

func newParams(req mcp.CallToolRequest) *params {
	return &params{req: req}
}

// require extracts a required string parameter. If the parameter is empty or missing,
// it is recorded and error() will return an appropriate error result.
func (p *params) require(name string) string {
	val := p.req.GetString(name, "")
	if val == "" {
		p.missing = append(p.missing, name)
	}
	return val
}

// optional extracts an optional string parameter, returning the default if missing.
func (p *params) optional(name, defaultVal string) string {
	return p.req.GetString(name, defaultVal)
}

// optionalBool extracts an optional boolean parameter.
func (p *params) optionalBool(name string, defaultVal bool) bool {
	return p.req.GetBool(name, defaultVal)
}

// error returns an MCP error result if any required parameters were missing, or nil if all are present.
func (p *params) error() *mcp.CallToolResult {
	if len(p.missing) == 0 {
		return nil
	}
	msg := fmt.Sprintf("missing required parameter: %s", strings.Join(p.missing, ", "))
	return mcp.NewToolResultError(msg)
}
