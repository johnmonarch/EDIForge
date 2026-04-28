package cli

import (
	"fmt"

	"github.com/johnmonarch/ediforge/internal/app"
)

func runVersion() error {
	fmt.Printf("%s %s\n", app.Command, app.Version)
	if app.Commit != "" && app.Commit != "unknown" {
		fmt.Printf("commit %s\n", app.Commit)
	}
	if app.Date != "" && app.Date != "unknown" {
		fmt.Printf("built %s\n", app.Date)
	}
	return nil
}
