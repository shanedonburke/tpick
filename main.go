package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"tpick/explorer"
	"tpick/help"
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
		if isHelpArg(arg) {
			help.PrintHelp()
			os.Exit(0)
		} else {
			dirArg = arg
		}
	default:
		fmt.Println("Error: Invalid arguments")
		fmt.Println()
		help.PrintHelp()
		os.Exit(0)
	}

	return processDirectoryArg(dirArg)
}

func processDirectoryArg(dirArg string) string {
	dirPath, err := filepath.Abs(dirArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get absolute path of %s: %v\n", dirArg, err)
		os.Exit(1)
	}

	fileInfo, err := os.Stat(dirPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "Error: Directory '%s' does not exist\n", dirPath)
		} else {
			fmt.Fprintf(os.Stderr, "Error: Failed to stat directory '%s': %v\n", dirPath, err)
		}
		os.Exit(1)
	}
	if !fileInfo.IsDir() {
		// The user specified a file, so use that file's directory
		dirPath = filepath.Dir(dirPath)
	}
	return dirPath
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help"
}
