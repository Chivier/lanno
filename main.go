package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	GetInfoFromFileSystem(".")

	p := tea.NewProgram(NewModel())
	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}
