package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/term"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var kLoginWidth = [6]int{5, 1, 4, 2, 2, 2}

const kFlexIndex = 1 // tag column is flexible
const (
	columnKeyFilename    = "filename"
	columnKeyIcons       = "icons"
	columnKeyTags        = "tags"
	columnKeyCreatedTime = "created_time"
	columnKeyUpdatedTime = "updated_time"
	columnKeyVisitedTime = "visited_time"
)

type FileModel struct {
	table table.Model
}

func (m FileModel) Init() tea.Cmd {
	return nil
}

func (m FileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m FileModel) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

type FileInfo struct {
	Name        string   `json:"name"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
}

type LannoFileData struct {
	FileInfo []FileInfo `json:"file_info"`
}

type CommandItem struct {
	path            string
	lastUpdatedTime string
	lastVisitedTime string
	createTime      string
}

type Item struct {
	path            string
	description     string
	tag             []string
	lastUpdatedTime string
	lastVisitedTime string
	createTime      string
}

func dateFormatDarwin(date string) string {
	// Parse the input date string using the expected format
	parsedTime, err := time.Parse("Jan 2 15:04:05 2006", date)
	if err != nil {
		// Handle any parsing errors
		return ""
	}

	// Format the parsed time to the target format
	formattedTime := parsedTime.Format("06/01/02 15:04:05")

	return formattedTime
}

func AddTagToFile(fileName string, tag string) {
	// Read the JSON data
	file, err := os.Open(".lanno.json")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// Parse JSON data
	var data LannoFileData
	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	// Add the tag to the appropriate file
	for i, fileInfo := range data.FileInfo {
		if fileInfo.Name == fileName {
			data.FileInfo[i].Tags = append(fileInfo.Tags, tag)
			break
		}
	}

	// Convert the updated data back into JSON
	updatedJSON, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error converting data to JSON:", err)
		return
	}

	// Write the updated JSON data back to the file
	err = os.WriteFile(".lanno.json", updatedJSON, 0644)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
}

func GetInfoFromAnnoFile(path string) []FileInfo {
	// Check if ther is a ".lanno.json" file in current folder
	file_path := path + "/.lanno.json"
	if _, err := os.Stat(file_path); os.IsNotExist(err) {
		// Create a new .lanno.json file on the path
		_, err := os.Create(file_path)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Read the JSON data
	file, err := os.Open(file_path)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return []FileInfo{}
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return []FileInfo{}
	}

	// Parse JSON data
	var data LannoFileData
	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return []FileInfo{}
	}

	// Access the parsed data
	fileInfo := data.FileInfo

	return fileInfo
}

func GoExecStatCommand(command string) string {
	// Execute command and get result
	cmd := exec.Command("/bin/sh", "-c", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	result := out.String()

	if runtime.GOOS == "linux" {
		// TODO format time on linux
	} else if runtime.GOOS == "darwin" {
		// Remove "\x0a" trailing space
		result = strings.TrimSpace(result)
		result = dateFormatDarwin(result)
	}
	return result
}

func GetInfoFromFileSystem(path string) CommandItem {
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
		//fmt.Println("Running on macOS")
		createdTimeCommand = "stat -f %SB " + path
		lastUpdatedTimeCommand = "stat -f %Sm " + path
		lastVisitedTimeCommand = "stat -f %Sa " + path
	} else {
		fmt.Println("Running on a different operating system")
	}

	return CommandItem{
		path:            path,
		lastUpdatedTime: GoExecStatCommand(lastUpdatedTimeCommand),
		lastVisitedTime: GoExecStatCommand(lastVisitedTimeCommand),
		createTime:      GoExecStatCommand(createdTimeCommand),
	}
}

func GetTableItems(path string) []table.Row {
	// Get information from sqlite
	lannoFileInfoList := GetInfoFromAnnoFile(path)

	// List out all files and folders in path, ignore hidden files
	files, err := os.ReadDir(path)

	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".") {
			continue // Skip hidden files
		}
	}

	var resultTable []table.Row
	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".") {
			continue // Skip hidden files
		}
		// Get information from file system
		commandItem := GetInfoFromFileSystem(path + "/" + file.Name())
		// Get information from anno
		lannoinfoItem := FileInfo{}
		for _, item := range lannoFileInfoList {
			if item.Name == file.Name() {
				lannoinfoItem = item
				break
			}
		}
		row := table.NewRow(table.RowData{
			columnKeyFilename:    file.Name(),
			columnKeyIcons:       "",
			columnKeyTags:        strings.Join(lannoinfoItem.Tags, ", "),
			columnKeyCreatedTime: commandItem.createTime,
			columnKeyUpdatedTime: commandItem.lastUpdatedTime,
			columnKeyVisitedTime: commandItem.lastVisitedTime,
		})

		resultTable = append(resultTable, row)
	}

	return resultTable
}

func NewModel() FileModel {
	// Get terminal width
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 80
	}

	border := 4
	columnNumber := 5
	tableWidth := make([]int, columnNumber)
	widthSum := 0
	for i := 0; i < columnNumber; i++ {
		widthSum += kLoginWidth[i]
	}
	for i := 0; i < columnNumber; i++ {
		tableWidth[i] = (width - border - columnNumber + 1) * kLoginWidth[i] / widthSum
	}
	tableWidth[kFlexIndex] = width - border - columnNumber + 1
	for i := 0; i < columnNumber; i++ {
		if i != kFlexIndex {
			tableWidth[kFlexIndex] -= tableWidth[i]
		}
	}

	columns := []table.Column{
		table.NewColumn(columnKeyFilename, "File/Folder Name", tableWidth[0]).WithFiltered(true),
		table.NewColumn(columnKeyIcons, "Icons", tableWidth[1]),
		table.NewColumn(columnKeyTags, "Tags", tableWidth[2]).WithFiltered(true),
		table.NewColumn(columnKeyCreatedTime, "Created Time", tableWidth[3]),
		table.NewColumn(columnKeyUpdatedTime, "Last Updated Time", tableWidth[4]),
		//table.NewColumn(columnKeyVisitedTime, "Last Visited Time", tableWidth[5]),
	}

	t := table.New(columns).
		Filtered(true).
		Focused(true).
		WithPageSize(30).
		WithRows(GetTableItems("."))

	return FileModel{
		t,
	}
}
