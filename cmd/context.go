package cmd

import (
	"context"

	"github.com/urfave/cli/v2"
)

// commandContext is the context passed to App.RunContext.
// The CLI process uses context.Background(), so each command keeps its own
// deadline. The mobile bridge passes a shorter context and that deadline wins.
func commandContext(c *cli.Context) context.Context {
	if c != nil && c.Context != nil {
		return c.Context
	}
	return context.Background()
}
