//go:build windows

package main

import (
	"os"

	"github.com/jiangwan0130/gemini-cli-notify/internal/notify"
)

func main() {
	os.Exit(notify.Run(notify.CodexTool(), os.Args[1:]))
}
