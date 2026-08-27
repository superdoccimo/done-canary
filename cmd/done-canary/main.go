package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/superdoccimo/done-canary/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(app.Execute(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
