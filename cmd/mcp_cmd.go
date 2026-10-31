package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
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
					"If the port is already in use, the process exits and does not pick another port or kill the holder. " +
					"mcp.pid and mcp.lock beside the config file keep a second serve from binding. " +
					"This process does not signal other pids. " +
					"The Mac app passes --parent-pid and --exit-on-stdin-close so the child exits when the app dies. " +
					"When mcp.require_token is set, both /mcp and /healthz require Authorization: Bearer from the mode-0600 token file. " +
					"The token is never printed. This command ships in the release wx binary.",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "enable",
						Usage: "listen even when mcp.enabled is false in config",
					},
					&cli.IntFlag{
						Name:  "port",
						Usage: "listen port; overrides mcp.port (default 9742)",
					},
					&cli.StringFlag{
						Name:  "config",
						Usage: "config.json path (default ~/.config/wx/config.json). The pidfile, lock, and default token file live beside it",
					},
					&cli.StringFlag{
						Name:  "token-file",
						Usage: "bearer token file (mode 0600, not a symlink). Overrides mcp.token_file. The token is not accepted on the command line",
					},
					&cli.IntFlag{
						Name:  "parent-pid",
						Usage: "exit when this pid dies (Mac supervisor). Must be > 1. The value is not signaled",
					},
					&cli.BoolFlag{
						Name:  "exit-on-stdin-close",
						Usage: "exit when stdin reaches EOF. The Mac app holds that pipe open and does not write to it",
					},
				},
				Action: runMCPServe,
			},
		},
	}
}

func runMCPServe(c *cli.Context) error {
	configPath := c.String("config")
	var cfg *config.Config
	var err error
	if configPath != "" {
		cfg, err = config.LoadFrom(configPath)
	} else {
		configPath, err = config.Path()
		if err != nil {
			return err
		}
		cfg, err = config.Load()
	}
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
	require, token, err := mcp.LoadServeAuth(configPath, cfg.MCP, c.String("token-file"))
	if err != nil {
		return err
	}
	opts.RequireToken = require
	opts.Token = token
	opts.RuntimeDir = filepath.Dir(configPath)
	opts.ExitOnStdinClose = c.Bool("exit-on-stdin-close")
	if c.IsSet("parent-pid") {
		parent := c.Int("parent-pid")
		if parent <= 1 {
			return fmt.Errorf("mcp: refusing --parent-pid %d", parent)
		}
		opts.ParentPID = parent
	}
	ctx, stop := signal.NotifyContext(commandContext(c), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := mcp.ListenAndServe(ctx, opts); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "wx mcp serve: stopped")
	return nil
}
