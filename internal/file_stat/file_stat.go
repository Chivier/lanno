package file_stat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	inputMode   bool
	inputPrompt string
	inputBuffer string
	inputTarget string
}

func (m FileModel) Init() tea.Cmd {
	return nil
}

func (m FileModel) View() string {
	view := baseStyle.Render(m.table.View())
	if m.searchMode {
		view += "\nSearch: " + m.searchQuery
	}
	if m.inputMode {
		view += "\n" + m.inputPrompt + m.inputBuffer
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
			return
		}
	}

	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		return
	}

	// Check if the JSON file is empty or has invalid format
	if len(byteValue) == 0 || !json.Valid(byteValue) {
		// Initialize with empty structure if file is empty or invalid
		byteValue = []byte("{}")
	}

	path = path[strings.LastIndex(path, "/")+1:]
	var data LannoFileData
	err = json.Unmarshal(byteValue, &data)
	if err != nil {
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

	if len(command) == 0 {
		description := ""
		for _, commandItem := range command {
			description += commandItem + " "
		}
		description = strings.TrimSpace(description)
		data.FileInfo[fileIndex].Description = description
		file, err = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return
		}
		byteValue, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return
		}
		_, err = file.Write(byteValue)
		if err != nil {
			return
		}
		defer file.Close()
		return
	}
	
	firstCommand := command[0]
	if firstCommand[0] != '+' && firstCommand[0] != '-' {
		description := ""
		for _, commandItem := range command {
			description += commandItem + " "
		}
		description = strings.TrimSpace(description)
		
		// If description is empty or just spaces, set it to empty string
		if description == "" {
			description = ""
		}
		data.FileInfo[fileIndex].Description = description
		file, err = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return
		}
		byteValue, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return
		}
		_, err = file.Write(byteValue)
		if err != nil {
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
	data.FileInfo[fileIndex].Tags = tagList
	file, err = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return
	}
	byteValue, err = json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	_, err = file.Write(byteValue)
	if err != nil {
		return
	}

	defer file.Close()
	return
}

func GetInfoFromAnnoFile(path string) map[string]FileInfo {
	filePath := path + "/.lanno.json"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		file, err := os.Create(filePath)
		file.WriteString("{}")
		file.Close()
		if err != nil {
			return map[string]FileInfo{}
		}
	}
	file, err := os.Open(filePath)
	if err != nil {
		return map[string]FileInfo{}
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		return map[string]FileInfo{}
	}
	var data LannoFileData
	err = json.Unmarshal(byteValue, &data)
	if err != nil {
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
		return ""
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
		// Running on a different operating system
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
		return []table.Row{}
	}
	
	// Get terminal width for proper truncation
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 80
	}
	
	// Calculate column widths based on terminal size
	availableWidth := width - 6
	nameWidth := (availableWidth * 30) / 100
	tagsWidth := (availableWidth * 20) / 100
	descWidth := availableWidth - nameWidth - tagsWidth
	
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
		
		// Use calculated widths for truncation
		filename := truncateText(icon+" "+file.Name(), nameWidth)
		tags := truncateText(strings.Join(lannoinfoItem.Tags, ", "), tagsWidth)
		desc := truncateText(lannoinfoItem.Description, descWidth)

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

// Add this type near the top of the file with other types
type refreshMsg struct{}

// Add this new type for screen clearing
type clearScreenMsg struct{}

func RefreshTableModel(m FileModel) FileModel {
	// Clear any existing state to prevent duplication
	m.table = nil
	
	// Get fresh data
	rows := GetTableItems(".")
	
	// Get current table properties
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 80
	}
	
	// Calculate column widths
	availableWidth := width - 6
	nameWidth := (availableWidth * 30) / 100
	tagsWidth := (availableWidth * 20) / 100
	descWidth := availableWidth - nameWidth - tagsWidth
	
	// Create new columns with same properties
	columns := []table.Column{
		table.NewColumn(columnKeyFilename, "Name", nameWidth).WithFiltered(true),
		table.NewColumn(columnKeyTags, "Tags", tagsWidth).WithFiltered(true),
		table.NewColumn(columnKeyDescription, "Description", descWidth),
	}
	// Calculate dynamic name column width based on max filename length
	maxNameWidth := 0
	for _, row := range rows {
		if filename, ok := row.Data[columnKeyFilename].(string); ok {
			nameLen := len(filename) + 1 // Add 1 for padding
			if nameLen > maxNameWidth {
				maxNameWidth = nameLen
			}
		}
	}
	
	// Ensure name column width is reasonable (not too small or too large)
	if maxNameWidth < 10 {
		maxNameWidth = 10 // Minimum width
	} else if maxNameWidth > nameWidth {
		maxNameWidth = nameWidth // Cap at original allocation
	}
	
	// Redistribute the space - give any saved space to description
	descWidth = availableWidth - maxNameWidth - tagsWidth
	
	// Update column widths
	columns[0].Width = maxNameWidth // Name column
	columns[2].Width = descWidth    // Description column

	// Create a completely new table
	t := table.New(columns).
		WithFiltered(true).
		WithFocused(true).
		WithPageSize(15).
		WithRows(rows)
	
	// Apply styles
	s := table.DefaultStyles()
	t.SetStyles(s)
	
	// Update model with new table and rows
	m.table = t
	m.allRows = rows
	
	return m
}

func (m FileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle clear screen message
	if _, ok := msg.(clearScreenMsg); ok {
		return RefreshTableModel(m), nil
	}

	// Handle refresh message
	if _, ok := msg.(refreshMsg); ok {
		// Store the current selection index before refreshing
		var selectedIndex int
		if len(m.table.SelectedRows()) > 0 {
			selectedIndex = m.table.Selected
		}
		
		// Refresh the model
		refreshedModel := RefreshTableModel(m)
		
		// Restore the selection if possible
		if selectedIndex >= 0 && selectedIndex < len(refreshedModel.table.Rows) {
			refreshedModel.table.Selected = selectedIndex
		}
		
		return refreshedModel, nil
	}

	// Handle window size changes
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		// First clear the screen, then rebuild the model
		return m, tea.Sequence(
			tea.ClearScreen,
			func() tea.Msg { return clearScreenMsg{} },
		)
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if m.inputMode {
			switch keyMsg.String() {
			case "enter":
				// Process the input
				command := strings.TrimSpace(m.inputBuffer)
				if command != "" {
					words := strings.Fields(command)
					TagCommand(words, m.inputTarget)
				} else {
					TagCommand([]string{}, m.inputTarget)
				}

				// Store the current selection index before exiting input mode
				selectedIndex := m.table.Selected
				
				m.inputMode = false
				m.inputBuffer = ""
				m.inputTarget = ""
				
				// Return a command to refresh the model after processing
				return m, func() tea.Msg { 
					// Store the selection index in the model before refreshing
					m.table.Selected = selectedIndex
					return refreshMsg{} 
				}
			case "esc":
				m.inputMode = false
				m.inputBuffer = ""
				m.inputTarget = ""
				// Also refresh when canceling input mode
				return m, func() tea.Msg { return refreshMsg{} }
			case "backspace", "ctrl+h":
				if len(m.inputBuffer) > 0 {
					m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
				}
				return m, nil
			default:
				// Append single-character keys to the input buffer
				if len(keyMsg.String()) == 1 {
					m.inputBuffer += keyMsg.String()
				}
				return m, nil
			}
		} else if m.searchMode {
			switch keyMsg.String() {
			case "enter":
				m.searchMode = false
				return m, func() tea.Msg { return refreshMsg{} }
			case "esc":
				m.searchMode = false
				m.searchQuery = ""
				// Reset rows to show all entries
				m.table = m.table.WithRows(m.allRows)
				return m, nil
			case "backspace", "ctrl+h":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				}
				m.table = m.table.WithRows(filterRows(m.allRows, m.searchQuery))
				return m, nil
			default:
				// Append single-character keys to the query.
				if len(keyMsg.String()) == 1 {
					m.searchQuery += keyMsg.String()
					m.table = m.table.WithRows(filterRows(m.allRows, m.searchQuery))
				}
				return m, nil
			}
		} else if keyMsg.String() == "/" {
			m.searchMode = true
			m.searchQuery = ""
			return m, nil
		}

		switch keyMsg.String() {
		case "ctrl+c", "q":
			cmds = append(cmds, tea.Quit)
		case "f5", "r":
			// Manual refresh
			return m, func() tea.Msg { return refreshMsg{} }
		case "ctrl+e":
			// Get the selected file
			if len(m.table.SelectedRows()) > 0 {
				selectedRow := m.table.SelectedRows()[0]
				if filename, ok := selectedRow.Data[columnKeyFilename].(string); ok {
					// Extract just the filename without icon
					parts := strings.SplitN(filename, " ", 2)
					if len(parts) > 1 {
						filename = parts[1]
					}
					// Remove ellipsis if present
					filename = strings.TrimSuffix(filename, "...")
					
					// Enter input mode
					m.inputMode = true
					m.inputPrompt = "Enter command for " + filename + ": "
					m.inputBuffer = ""
					m.inputTarget = filename
					return m, nil
				}
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

func filterRows(rows []table.Row, query string) []table.Row {
	var filtered []table.Row
	lowerQuery := strings.ToLower(query)
	for _, row := range rows {
		name := ""
		tags := ""
		desc := ""
		if val, ok := row.Data[columnKeyFilename]; ok {
			name = fmt.Sprintf("%v", val)
		}
		if val, ok := row.Data[columnKeyTags]; ok {
			tags = fmt.Sprintf("%v", val)
		}
		if val, ok := row.Data[columnKeyDescription]; ok {
			desc = fmt.Sprintf("%v", val)
		}
		if strings.Contains(strings.ToLower(name), lowerQuery) || 
		   strings.Contains(strings.ToLower(tags), lowerQuery) || 
		   strings.Contains(strings.ToLower(desc), lowerQuery) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}
