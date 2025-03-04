package file_stat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/term"

	"lanno/internal/table"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder())

var kLoginWidth = [6]int{5, 1, 4, 6, 0, 0}

const kFlexIndex = 1 // tag column is flexible
const (
	columnKeyFilename    = "filename"
	columnKeyIcons       = "icons"
	columnKeyTags        = "tags"
	columnKeyDescription = "description"
	// columnKeyCreatedTime = "created_time"
	// columnKeyUpdatedTime = "updated_time"
	// columnKeyVisitedTime = "visited_time"
)

type FileModel struct {
	table       *table.Table
	searchMode  bool
	searchQuery string
	allRows     []table.Row
}

func (m FileModel) Init() tea.Cmd {
	return nil
}

func (m FileModel) View() string {
	view := baseStyle.Render(m.table.View())
	if m.searchMode {
		view += "\nSearch: " + m.searchQuery
	}
	return view + "\n"
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
	parsedTime, err := time.Parse("Jan 2 15:04:05 2006", date)
	if err != nil {
		return ""
	}
	formattedTime := parsedTime.Format("06/01/02 15:04:05")
	return formattedTime
}

func TagCommand(command []string, path string) {
	filePath := "./.lanno.json"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		_, err := os.Create(filePath)
		if err != nil {
			log.Fatal(err)
		}
	}

	file, err := os.Open(filePath)
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

	path = path[strings.LastIndex(path, "/")+1:]
	var data LannoFileData
	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	found := false
	fileIndex := -1
	var tagList []string
	for i, item := range data.FileInfo {
		if item.Name == path {
			tagList = item.Tags
			found = true
			fileIndex = i
			break
		}
	}

	if !found {
		fileIndex = len(data.FileInfo)
		data.FileInfo = append(data.FileInfo, FileInfo{Name: path, Tags: []string{}, Description: ""})
	}

	firstCommand := command[0]
	println("firstCommand: ", firstCommand)
	if firstCommand[0] != '+' && firstCommand[0] != '-' {
		description := ""
		for _, commandItem := range command {
			description += commandItem + " "
		}
		println("description: ", description)
		data.FileInfo[fileIndex].Description = description
		file, err = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		byteValue, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			fmt.Println("Error parsing JSON:", err)
			return
		}
		_, err = file.Write(byteValue)
		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}
		defer file.Close()
		return
	}
	for _, tagCommand := range command {
		tagString := "#" + tagCommand[1:]
		if tagCommand[0] == '+' {
			tagList = append(tagList, tagString)
		} else if tagCommand[0] == '-' {
			for i, tag := range tagList {
				if tag == tagString {
					tagList = append(tagList[:i], tagList[i+1:]...)
					break
				}
			}
		}
	}
	fmt.Println(tagList)
	data.FileInfo[fileIndex].Tags = tagList
	file, err = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	byteValue, err = json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}
	_, err = file.Write(byteValue)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return
	}

	defer file.Close()
	return
}

func GetInfoFromAnnoFile(path string) map[string]FileInfo {
	filePath := path + "/.lanno.json"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		println("create file")
		file, err := os.Create(filePath)
		file.WriteString("{}")
		file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return map[string]FileInfo{}
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return map[string]FileInfo{}
	}
	var data LannoFileData
	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return map[string]FileInfo{}
	}
	fileInfoMap := make(map[string]FileInfo)
	for _, item := range data.FileInfo {
		if strings.HasPrefix(item.Name, "./") {
			item.Name = item.Name[2:]
		}
		fileInfoMap[item.Name] = item
	}
	return fileInfoMap
}

func GoExecStatCommand(command string) string {
	cmd := exec.Command("/bin/sh", "-c", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	result := out.String()
	if runtime.GOOS == "linux" {
		// TODO: format time on linux
	} else if runtime.GOOS == "darwin" {
		result = strings.TrimSpace(result)
		result = dateFormatDarwin(result)
	}
	return result
}

func GetInfoFromFileSystem(path string) CommandItem {
	createdTimeCommand := ""
	lastUpdatedTimeCommand := ""
	lastVisitedTimeCommand := ""

	if runtime.GOOS == "linux" {
		createdTimeCommand = "stat -c %w " + path
		lastUpdatedTimeCommand = "stat -c %y " + path
		lastVisitedTimeCommand = "stat -c %x " + path
	} else if runtime.GOOS == "darwin" {
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
	lannoInfoMap := GetInfoFromAnnoFile(path)
	files, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	var resultTable []table.Row
	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".") {
			continue
		}
		lannoinfoItem := lannoInfoMap[file.Name()]
		icon := "📄"
		if file.IsDir() {
			icon = "📁"
		}
		
		// Truncate without column widths for now
		filename := truncateText(icon+" "+file.Name(), 30)  // reasonable default
		tags := truncateText(strings.Join(lannoinfoItem.Tags, ", "), 20)
		desc := truncateText(lannoinfoItem.Description, 50)
		
		row := table.NewRow(table.RowData{
			columnKeyFilename:    filename,
			columnKeyTags:        tags,
			columnKeyDescription: desc,
		})
		resultTable = append(resultTable, row)
	}
	return resultTable
}

func NewModel() FileModel {
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 80
	}
	
	// Calculate column widths
	// Name 30%, Tags 20%, Description 50% of available width
	// Account for borders (2 vertical lines = 2 chars)
	availableWidth := width - 6
	nameWidth := (availableWidth * 30) / 100
	tagsWidth := (availableWidth * 20) / 100
	descWidth := availableWidth - nameWidth - tagsWidth // Use remaining space
	
	columns := []table.Column{
		table.NewColumn(columnKeyFilename, "Name", nameWidth).WithFiltered(true),
		table.NewColumn(columnKeyTags, "Tags", tagsWidth).WithFiltered(true),
		table.NewColumn(columnKeyDescription, "Description", descWidth),
	}
	
	rows := GetTableItems(".")
	t := table.New(columns).
		WithFiltered(true).
		WithFocused(true).
		WithPageSize(15).
		WithRows(rows)
	
	s := table.DefaultStyles()
	t.SetStyles(s)
	
	return FileModel{
		table:   t,
		allRows: rows,
	}
}

// Helper function to truncate text with ellipsis
func truncateText(text string, width int) string {
	if width <= 3 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	return string(runes[:width-3]) + "..."
}

func (m FileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if m.searchMode {
			switch keyMsg.String() {
			case "enter":
				m.searchMode = false
			case "esc":
				m.searchMode = false
				m.searchQuery = ""
				// Reset rows to show all entries.
				m.table.WithRows(m.allRows)
			case "backspace", "ctrl+h":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				}
				m.table.WithRows(filterRows(m.allRows, m.searchQuery))
			default:
				// Append single-character keys to the query.
				if len(keyMsg.String()) == 1 {
					m.searchQuery += keyMsg.String()
					m.table.WithRows(filterRows(m.allRows, m.searchQuery))
				}
			}
			return m, tea.Batch(cmds...)
		} else if keyMsg.String() == "/" {
			m.searchMode = true
			m.searchQuery = ""
			return m, nil
		}

		switch keyMsg.String() {
		case "ctrl+c", "q":
			cmds = append(cmds, tea.Quit)
		case "ctrl+e":
			selected := m.table.SelectedRows()
			if len(selected) > 0 {
				filename := selected[0].Data[columnKeyFilename].(string)
				cmds = append(cmds, launchEditor(filename))
			}
		}
	}

	newTable, cmd := m.table.Update(msg)
	m.table = newTable
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func launchEditor(filename string) tea.Cmd {
	return func() tea.Msg {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}
		cmd := exec.Command(editor, filename)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			log.Printf("Error running editor: %v", err)
		}

		return nil
	}
}

func filterRows(rows []table.Row, query string) []table.Row {
	var filtered []table.Row
	lowerQuery := strings.ToLower(query)
	for _, row := range rows {
		name := ""
		desc := ""
		if val, ok := row.Data[columnKeyFilename]; ok {
			name = fmt.Sprintf("%v", val)
		}
		if val, ok := row.Data[columnKeyDescription]; ok {
			desc = fmt.Sprintf("%v", val)
		}
		if strings.Contains(strings.ToLower(name), lowerQuery) || strings.Contains(strings.ToLower(desc), lowerQuery) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}
