package main

import (
	"fmt"
	"os"
)

func ExitIfError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
