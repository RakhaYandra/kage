package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/RakhaYandra/kage/internal/cli"
)

func main() {
	root := cli.New()
	if err := root.Execute(); err != nil {
		var ee *cli.ExitError
		if errors.As(err, &ee) {
			fmt.Fprintln(os.Stderr, "kage: "+ee.Msg)
			os.Exit(ee.Code)
		}
		fmt.Fprintln(os.Stderr, "kage: "+err.Error())
		os.Exit(1)
	}
}
