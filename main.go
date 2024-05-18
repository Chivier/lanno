package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"log"
	"os"
)

func view() {
	p := tea.NewProgram(NewModel())
	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	// parse parameters
	if len(os.Args) < 2 {
		view()
	} else {
		filePath := os.Args[1]
		tagEditCommand := os.Args[2:]
		TagCommand(tagEditCommand, filePath)
	}
}
