package main

import (
	"fmt"
	"log"
	"os"

	"lanno/internal/file_stat"

	tea "github.com/charmbracelet/bubbletea"
)

const helpText = `lanno - A file tagging and organization tool

Usage:
    lanno                    # Launch interactive file browser
    lanno <file> <command>   # Tag or describe a file

Commands:
    +<tag>                   # Add a tag to a file
    -<tag>                   # Remove a tag from a file
    <description>            # Set description for a file

Examples:
    lanno document.txt +work     # Add #work tag to document.txt
    lanno document.txt -work     # Remove #work tag from document.txt
    lanno document.txt "Important work document"  # Set description
    lanno document.txt +urgent "Important work document"  # Add tag and description

Interactive Mode:
    /              # Search files
    ctrl+e         # Edit selected file tags or description, +<tag>, -<tag>, or <description>
    q or ctrl+c    # Quit
`

// Version represents the current application version
const Version = "1.0.0"

func printHelp() {
	fmt.Print(helpText)
	os.Exit(0)
}

func view() {
	p := tea.NewProgram(file_stat.NewModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	// Check for version flag
	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Printf("lanno v%s\n", Version)
		os.Exit(0)
	}

	// Check for help flags
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "--help" || arg == "-h" {
			printHelp()
		}
	}

	// parse parameters
	if len(os.Args) < 2 {
		view()
	} else {
		filePath := os.Args[1]
		tagEditCommand := os.Args[2:]
		file_stat.TagCommand(tagEditCommand, filePath)
	}
}
