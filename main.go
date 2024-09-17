package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"tpick/explorer"
	"tpick/explorer/help"
	"tpick/screen"

	"github.com/gdamore/tcell/v2"
)

func main() {
	directory := processArgs()

	s := screen.InitScreen()
	e := explorer.NewExplorer(s, directory)

	for {
		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			e.HandleResize()
		case *tcell.EventKey:
			e.HandleKeyEvent(ev)
		}
	}
}

func processArgs() string {
	var dirArg string
	numArgs := len(os.Args)

	switch numArgs {
	case 1:
		dirArg = "."
	case 2:
		arg := strings.TrimSpace(os.Args[1])
		if arg == "-h" || arg == "--help" {
			help.PrintHelp()
			os.Exit(0)
		} else {
			dirArg = arg
		}
	default:
		help.PrintHelp()
		os.Exit(0)
	}

	absDir, err := filepath.Abs(dirArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get absolute path of %s: %v\n", dirArg, err)
		os.Exit(1)
	}

	return absDir
}
