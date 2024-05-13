package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"log"
)

func main() {
	GetInfoFromFileSystem(".")
	p := tea.NewProgram(NewModel())

	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}
