package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"os"
	"runtime"

	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)



var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))
	

const (
	columnKeyTitle       = "title"
	columnKeyAuthor      = "author"
	columnKeyDescription = "description"
)

type Model struct {
	table table.Model
}

type sqlItem struct {
	path string
	description string
	tag []string
}

type commandItem struct {
	path string
	lastUpdatedTime string
    lastVisitedTime string
	createTime string
}

type Item struct {
	path string
	description string
	tag []string
	lastUpdatedTime string
    lastVisitedTime string
	createTime string
}

func GetInfoFromSqlite (path string) sqlItem{
	// Search string in a sqlite file in ~/.config/lanno/db.sqlite3
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	dbPath := home + "/.config/lanno/db.sqlite3"
	db, err := sql.Open("sqlite3", dbPath)
	// if database not exist, create it
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Search path in the sqlite file
	sqlStatement := `SELECT path, description, tag FROM items WHERE path = ?;`
	row := db.QueryRow(sqlStatement, path)
	var item sqlItem
	err = row.Scan(&item.path, &item.description, &item.tag)
	if err != nil {
		log.Fatal(err)
	}
	return item
}

// func DateFormat(dateinfo string) string{
// 	if runtime.GOOS == "linux" {

// }

func GoExecStatCommand(command string) string{
	// Execute command and get result
	cmd := exec.Command("/bin/sh", "-c", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	result := out.String()
	println(result)
	return result
}

func GetInfoFromFileSystem (path string) commandItem{
	// Search string in a sqlite file in ~/.config/lanno/db.sqlite3
	home, err := os.UserHomeDir()
	println(home)
	if err != nil {
		log.Fatal(err)
	}

	createdTimeCommand := ""
	lastUpdatedTimeCommand := ""
	lastVisitedTimeCommand := ""
	// Get platform, mac, linux or windows
	if runtime.GOOS == "linux" {
		// fmt.Println("Running on Linux")
		createdTimeCommand = "stat -c %w" + path
		lastUpdatedTimeCommand = "stat -c %y" + path
		lastVisitedTimeCommand = "stat -c %x" + path
	} else if runtime.GOOS == "darwin" {
		fmt.Println("Running on macOS")
		createdTimeCommand = "stat -f %SB " + path
		lastUpdatedTimeCommand = "stat -f %Sm " + path
		lastVisitedTimeCommand = "stat -f %Sa " + path
	} else {
		fmt.Println("Running on a different operating system")
	}

	GoExecStatCommand(createdTimeCommand)
	GoExecStatCommand(lastUpdatedTimeCommand)
	GoExecStatCommand(lastVisitedTimeCommand)
	return commandItem{
		path: path,
		lastUpdatedTime: GoExecStatCommand(lastUpdatedTimeCommand),
		lastVisitedTime: GoExecStatCommand(lastVisitedTimeCommand),
		createTime: GoExecStatCommand(createdTimeCommand),
	}

	// GetInfoFromFileSystem
}

func NewModel() Model {
	columns := []table.Column{
		table.NewColumn(columnKeyTitle, "Title", 13).WithFiltered(true),
		table.NewColumn(columnKeyAuthor, "Author", 13).WithFiltered(true),
		table.NewColumn(columnKeyDescription, "Description", 50),
	}

	t := table.New(columns).
	Filtered(true).
	Focused(true).
	WithPageSize(10).
	WithRows([]table.Row{
		table.NewRow(table.RowData{
			columnKeyTitle:       "Computer Systems : A Programmer's Perspective",
			columnKeyAuthor:      "Randal E. Bryant、David R. O'Hallaron / Prentice Hall ",
			columnKeyDescription: "This book explains the important and enduring concepts underlying all computer...",
		}),
		table.NewRow(table.RowData{
			columnKeyTitle:       "Effective Java : 3rd Edition",
			columnKeyAuthor:      "Joshua Bloch",
			columnKeyDescription: "The Definitive Guide to Java Platform Best Practices—Updated for Java 9 Java ...",
		}),
		table.NewRow(table.RowData{
			columnKeyTitle:       "Structure and Interpretation of Computer Programs - 2nd Edition (MIT)",
			columnKeyAuthor:      "Harold Abelson、Gerald Jay Sussman",
			columnKeyDescription: "Structure and Interpretation of Computer Programs has had a dramatic impact on...",
		}),
		table.NewRow(table.RowData{
			columnKeyTitle:       "Game Programming Patterns",
			columnKeyAuthor:      "Robert Nystrom / Genever Benning",
			columnKeyDescription: "The biggest challenge facing many game programmers is completing their game. M...",
		}),
	})

	return Model{
		t,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			cmds = append(cmds, tea.Quit)
		}

	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {

	return baseStyle.Render(m.table.View()) + "\n"
}

func main() {
	GetInfoFromFileSystem(".")
	p := tea.NewProgram(NewModel())

	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}
