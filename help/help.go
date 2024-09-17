package help

import (
	"fmt"
	"strings"
)

const HELP = `
tpick - Terminal file picker

Usage: tpick [directory]

Arguments:
  [directory]   (Optional) Starting directory

Options:
  -h, --help    Show this help message and exit

Examples:
  tpick
  tpick /home/me
`

func PrintHelp() {
	fmt.Print(strings.TrimSpace(HELP))
}
