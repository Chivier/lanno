package main

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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
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
	//columnKeyCreatedTime = "created_time"
	//columnKeyUpdatedTime = "updated_time"
	//columnKeyVisitedTime = "visited_time"
)

type FileModel struct {
	table table.Model
}

func (m FileModel) Init() tea.Cmd {
	return nil
}

// func (m FileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
// 	var (
// 		cmd  tea.Cmd
// 		cmds []tea.Cmd
// 	)

// 	switch msg := msg.(type) {
// 	case tea.KeyMsg:
// 		switch msg.String() {
// 		case "ctrl+c", "q":
// 			cmds = append(cmds, tea.Quit)
// 		case "e":
// 			// Get the selected row
// 			if selected := m.table.SelectedRows(); len(selected) > 0 {
// 				filename := selected[0].Data[columnKeyFilename].(string)
// 				// Launch external editor for the selected file
// 				cmds = append(cmds, launchEditor(filename))
// 			}
// 		}
// 	}

// 	// Move table update after our custom key handling
// 	m.table, cmd = m.table.Update(msg)
// 	cmds = append(cmds, cmd)

// 	return m, tea.Batch(cmds...)
// }

func (m FileModel) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
	// Safety valve for empty tables
}

//	func (m FileModel) View() string {
//		var b strings.Builder
//
//		for i, row := range m.table.Rows() {
//			style := baseStyle
//
//			// Check if the row is selected
//			if m.table.IsSelected(i) {
//				// If the row is selected, apply bold font and yellow color
//				style = style.Bold(true).Foreground(lipgloss.Color("3"))
//			}
//
//			// Render the row with the applied style
//			b.WriteString(style.Render(row.View()))
//			b.WriteRune('\n')
//		}
//
//		return b.String()
//	}
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

func TagCommand(command []string, path string) {
	filePath := "./.lanno.json"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Create a new .lanno.json file on the path
		_, err := os.Create(filePath)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Read the JSON data
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

	// Convert path to name
	path = path[strings.LastIndex(path, "/")+1:]

	// Parse JSON data
	var data LannoFileData
	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	// if the file is not in the list, add it
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

	// Chech if this is a tag command for update description
	firstCommand := command[0]
	println("firstCommand: ", firstCommand)
	if firstCommand[0] != '+' && firstCommand[0] != '-' {
		// Concatenate all the commands to get the description
		description := ""
		for _, commandItem := range command {
			description += commandItem + " "
		}
		println("description: ", description)
		// save the description
		data.FileInfo[fileIndex].Description = description
		// save data back to the file
		file, err = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		// Write the JSON data
		byteValue, err = json.Marshal(data)
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
			// if the tag is int the list, remove it
			for i, tag := range tagList {
				if tag == tagString {
					tagList = append(tagList[:i], tagList[i+1:]...)
					break
				}
			}
		}
	}
	// print the tag list
	fmt.Println(tagList)

	// save the tag list
	data.FileInfo[fileIndex].Tags = tagList
	// save data back to the file
	file, err = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	// Write the JSON data
	byteValue, err = json.Marshal(data)
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
	// Check if ther is a ".lanno.json" file in current folder
	filePath := path + "/.lanno.json"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Create a new .lanno.json file on the path
		// Write {} to empty json file
		println("create file")
		file, err := os.Create(filePath)
		file.WriteString("{}")
		file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}

	// Read the JSON data
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

	// Parse JSON data
	var data LannoFileData
	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return map[string]FileInfo{}
	}

	// Access the parsed data
	fileInfoMap := make(map[string]FileInfo)
	for _, item := range data.FileInfo {
		// if item.Name begin with "./", remove it
		if strings.HasPrefix(item.Name, "./") {
			item.Name = item.Name[2:]
		}
		fileInfoMap[item.Name] = item
	}

	return fileInfoMap
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
	lannoInfoMap := GetInfoFromAnnoFile(path)

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
		//commandItem := GetInfoFromFileSystem(path + "/" + file.Name())
		// Get information from anno
		lannoinfoItem := lannoInfoMap[file.Name()]

		// if this is a file, use the file icon
		// if this is a folder, use the folder icon
		// if this is a link, use the link icon

		icon := ""
		if file.IsDir() {
			icon = "📁"
		} else {
			icon = "📄"
		}

		//println("tags: ", strings.Join(lannoinfoItem.Tags, ", "))
		row := table.NewRow(table.RowData{
			columnKeyFilename:    file.Name(),
			columnKeyIcons:       icon,
			columnKeyTags:        strings.Join(lannoinfoItem.Tags, ", "),
			columnKeyDescription: lannoinfoItem.Description,
			//columnKeyCreatedTime: commandItem.createTime,
			//columnKeyUpdatedTime: commandItem.lastUpdatedTime,
			//columnKeyVisitedTime: commandItem.lastVisitedTime,
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
		table.NewColumn(columnKeyFilename, "Name", tableWidth[0]).WithFiltered(true),
		table.NewColumn(columnKeyIcons, "Icons", tableWidth[1]),
		table.NewColumn(columnKeyTags, "Tags", tableWidth[2]).WithFiltered(true),
		table.NewColumn(columnKeyDescription, "Description", tableWidth[3]),
		//table.NewColumn(columnKeyCreatedTime, "Created Time", tableWidth[3]),
		//table.NewColumn(columnKeyUpdatedTime, "Last Updated Time", tableWidth[4]),
		//table.NewColumn(columnKeyVisitedTime, "Last Visited Time", tableWidth[5]),
	}

	t := table.New(columns).
		Filtered(true).
		Focused(true).
		WithPageSize(30).
		WithRows(GetTableItems(".")).
		WithKeyMap(table.DefaultKeyMap())

	return FileModel{
		t,
	}
}

func (m FileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var (
        cmd  tea.Cmd
        cmds []tea.Cmd
    )

    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            cmds = append(cmds, tea.Quit)
        case "e":
            // Get the selected row
            selected := m.table.SelectedRows()
            if selected != nil {
                filename := selected[0].Data[columnKeyFilename].(string)
                // Launch external editor for the selected file
                cmds = append(cmds, launchEditor(filename))
            }
        }
    }

    // Handle table updates
    newTable, cmd := m.table.Update(msg)
    m.table = newTable
    if cmd != nil {
        cmds = append(cmds, cmd)
    }

    return m, tea.Batch(cmds...)
}

func launchEditor(filename string) tea.Cmd {
    return func() tea.Msg {
        // Read current .lanno.json
        data := GetInfoFromAnnoFile(".")
        fileInfo := data[filename]
        
        // Create temporary file with current info
        tmpFile, err := os.CreateTemp("", "lanno-*.json")
        if err != nil {
            log.Printf("Error creating temp file: %v", err)
            return nil
        }
        defer os.Remove(tmpFile.Name())

        // Write current file info to temp file
        tempFileInfo := map[string]interface{}{
            "name": filename,
            "tags": fileInfo.Tags,
            "description": fileInfo.Description,
        }
        
        jsonData, err := json.MarshalIndent(tempFileInfo, "", "    ")
        if err != nil {
            log.Printf("Error marshaling JSON: %v", err)
            return nil
        }
        
        if _, err := tmpFile.Write(jsonData); err != nil {
            log.Printf("Error writing to temp file: %v", err)
            return nil
        }
        tmpFile.Close()

        // Get editor from environment or default to vim
        editor := os.Getenv("EDITOR")
        if editor == "" {
            editor = "vim"
        }

        // Launch editor
        cmd := exec.Command(editor, tmpFile.Name())
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        
        if err := cmd.Run(); err != nil {
            log.Printf("Error running editor: %v", err)
            return nil
        }

        // Read updated content
        content, err := os.ReadFile(tmpFile.Name())
        if err != nil {
            log.Printf("Error reading updated file: %v", err)
            return nil
        }

        // Parse updated content
        var updatedInfo struct {
            Name string   `json:"name"`
            Tags []string `json:"tags"`
            Description string `json:"description"`
        }
        if err := json.Unmarshal(content, &updatedInfo); err != nil {
            log.Printf("Error parsing updated content: %v", err)
            return nil
        }

        // Update .lanno.json
        commands := []string{updatedInfo.Description}
        for _, tag := range updatedInfo.Tags {
            if !strings.HasPrefix(tag, "#") {
                commands = append(commands, "+"+strings.TrimPrefix(tag, "#"))
            } else {
                commands = append(commands, "+"+strings.TrimPrefix(tag, "#"))
            }
        }
        TagCommand(commands, filename)

        return nil
    }
}
