package cmd

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCLIRejectsInvalidLatLon(t *testing.T) {
	app := NewApp()
	for _, loc := range []string{"91,0", "foo,bar", "41.07,", "41.07 -85.14"} {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := app.RunContext(ctx, []string{"wx", "--json", "-l", loc})
		cancel()
		if err == nil {
			t.Fatalf("wx -l %q succeeded", loc)
		}
		if !strings.Contains(err.Error(), "invalid location") {
			t.Fatalf("wx -l %q err = %v, want invalid location", loc, err)
		}
	}
}
