package cli

import (
	"fmt"

	"github.com/openedi/ediforge/internal/app"
)

func runVersion() error {
	fmt.Printf("%s %s\n", app.Command, app.Version)
	return nil
}
