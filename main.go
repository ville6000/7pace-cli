// Command 7pace-cli posts worklogs to an on-prem 7pace Timetracker.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/ville6000/7pace-cli/cmd"
)

func main() {
	// Cancel in-flight API requests on Ctrl-C.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := cmd.Execute(ctx)
	stop()

	if err != nil {
		os.Exit(1)
	}
}
