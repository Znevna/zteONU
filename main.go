package main

import (
	"os"

	"github.com/Znevna/zteONU/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
