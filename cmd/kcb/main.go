package main

import (
	"os"

	"kope.io/build/pkg/cmd"
)

func main() {
	cmd.Execute(os.Stdout)
}
