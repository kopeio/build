package main

import (
	"kope.io/imagebuilder/pkg/cmd"
	"os"
)

func main() {
	cmd.Execute(os.Stdout)
}
