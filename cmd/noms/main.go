package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/dotBeeps/noms/internal/ui"
)

// Version is the application version, set at build time or defaulting to dev.
var Version = "0.1.0-dev"

func main() {
	// Check for --version / -v flag
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-v" {
			fmt.Printf("noms v%s\n", Version)
			os.Exit(0)
		}
	}

	app := ui.NewApp()
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
