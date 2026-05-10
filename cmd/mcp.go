package cmd

import (
	"log"

	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/walter/p/internal/mcpserver"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run as an MCP server (stdio transport)",
	Long: `Start p as an MCP server over stdio. This is used internally when
p spawns a claude subprocess — claude connects to this server to
call the deterministic edit primitives.

Can also be used standalone with any MCP-compatible client.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		s := mcpserver.NewServer(cfg.ProjectRoot)
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("MCP server error: %v", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
