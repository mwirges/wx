package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/mwirges/wx/internal/config"
	"github.com/mwirges/wx/internal/mcp"
)

func mcpCommand() *cli.Command {
	return &cli.Command{
		Name:  "mcp",
		Usage: "local Model Context Protocol server (loopback only)",
		Subcommands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "serve Streamable HTTP on 127.0.0.1 (default port 9742)",
				Description: "Listens on 127.0.0.1 only. Off unless mcp.enabled is true in config or --enable is set. " +
					"If the port is already in use, the process exits and does not pick another port. " +
					"This command ships in the release wx binary.",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "enable",
						Usage: "listen even when mcp.enabled is false in config",
					},
					&cli.IntFlag{
						Name:  "port",
						Usage: "listen port; overrides mcp.port (default 9742)",
					},
				},
				Action: runMCPServe,
			},
		},
	}
}

func runMCPServe(c *cli.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	opts, err := mcp.Prepare(mcp.ServeConfig{
		ConfigEnabled:   cfg.MCP.Enabled,
		FlagEnable:      c.Bool("enable"),
		ConfigPort:      cfg.MCP.Port,
		FlagPort:        c.Int("port"),
		FlagPortSet:     c.IsSet("port"),
		Version:         Version,
		DefaultLocation: cfg.DefaultLocation,
	})
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(commandContext(c), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := mcp.ListenAndServe(ctx, opts); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "wx mcp serve: stopped")
	return nil
}
