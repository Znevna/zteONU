package main

import (
	"log"

	"github.com/Znevna/zteONU/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Panicln(err)
	}
}
