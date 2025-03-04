package table

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Column represents a single table column.
type Column struct {
	Key      string
	Title    string
	Width    int
	Filtered bool
}

// NewColumn creates a new column.
func NewColumn(key, title string, width int) Column {
	return Column{Key: key, Title: title, Width: width}
}

// WithFiltered is a chainable method to mark the column as filtered.
func (c Column) WithFiltered(filtered bool) Column {
	c.Filtered = filtered
	return c
}

// RowData is a convenient alias for a map representing row data.
type RowData map[string]interface{}

// Row represents a single table row.
type Row struct {
	Data RowData
}

// NewRow creates a new row from the given RowData.
func NewRow(data RowData) Row {
	return Row{Data: data}
}

// Add these style-related types and constants
type Styles struct {
	Header   lipgloss.Style
	Selected lipgloss.Style
	Normal   lipgloss.Style
	Border   lipgloss.Style
}

func DefaultStyles() Styles {
	return Styles{
		Border: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")),
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252")),
		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("90")),
		Normal: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
	}
}

// Table represents the table model.
type Table struct {
	Columns  []Column
	Rows     []Row
	PageSize int
	Selected int
	focused  bool
	filtered bool
	styles   Styles
}

// New creates a new table instance with the provided columns.
func New(columns []Column) *Table {
	return &Table{
		Columns:  columns,
		Rows:     []Row{},
		PageSize: 10,
		Selected: 0,
		focused:  false,
		filtered: false,
	}
}

// WithPageSize sets the number of rows visible per page.
func (t *Table) WithPageSize(ps int) *Table {
	t.PageSize = ps
	return t
}

func (t *Table) WithFiltered(filtered bool) *Table {
	t.filtered = filtered
	return t
}

func (t *Table) WithFocused(focused bool) *Table {
	t.focused = focused
	return t
}

// WithRows sets the rows of the table.
func (t *Table) WithRows(rows []Row) *Table {
	t.Rows = rows
	return t
}

// WithKeyMap is provided for chaining but not used in this implementation.
func (t *Table) WithKeyMap(keyMap interface{}) *Table {
	return t
}

// SetFiltered toggles whether the table has filtering enabled.
func (t *Table) SetFiltered(enabled bool) *Table {
	t.filtered = enabled
	return t
}

// SetFocused toggles whether the table currently has focus.
func (t *Table) SetFocused(enabled bool) *Table {
	t.focused = enabled
	return t
}

// SetStyles adds styles to the table
func (t *Table) SetStyles(s Styles) *Table {
	t.styles = s
	return t
}

// Update handles key events to navigate through the table rows.
func (t *Table) Update(msg tea.Msg) (*Table, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if t.Selected > 0 {
				t.Selected--
			}
		case "down", "j":
			if t.Selected < len(t.Rows)-1 {
				t.Selected++
			}
		}
	}
	return t, nil
}

// SelectedRows returns the currently selected row(s) as a slice.
func (t *Table) SelectedRows() []Row {
	if len(t.Rows) == 0 {
		return []Row{}
	}
	return []Row{t.Rows[t.Selected]}
}

// View renders the table as a string.
func (t *Table) View() string {
	var b strings.Builder
	
	// Create the header with proper width and alignment
	headerRow := ""
	for i, col := range t.Columns {
		if i > 0 {
			headerRow += "│"
		}
		// Adjust width consistently with other rows
		title := fmt.Sprintf("%-*s", col.Width, col.Title)
		if len(title) > col.Width {
			title = title[:col.Width-3] + "..."
		}
		headerRow += title
	}
	b.WriteString(t.styles.Header.Render(headerRow))
	
	// Add separator line with intersections
	b.WriteString("\n")
	separatorLine := ""
	for i := 0; i < len(t.Columns); i++ {
		if i > 0 {
			separatorLine += "┼" + strings.Repeat("─", t.Columns[i].Width)
		} else {
			separatorLine += strings.Repeat("─", t.Columns[i].Width)
		}
	}
	b.WriteString(separatorLine)
	
	// Render rows
	for i, row := range t.Rows {
		b.WriteString("\n")
		rowContent := ""
		for j, col := range t.Columns {
			if j > 0 {
				rowContent += "│"
			}
			cell := ""
			if val, ok := row.Data[col.Key]; ok {
				cell = fmt.Sprintf("%v", val)
				
				// Calculate visual width accounting for special characters
				visualWidth := runewidth.StringWidth(cell)
				
				// Truncate if needed based on visual width
				if visualWidth > col.Width {
					if col.Width > 3 {
						// Truncate carefully considering visual width
						truncated := ""
						currentWidth := 0
						for _, r := range cell {
							charWidth := runewidth.RuneWidth(r)
							if currentWidth + charWidth + 3 > col.Width {
								break
							}
							truncated += string(r)
							currentWidth += charWidth
						}
						cell = truncated + "..."
					} else {
						cell = strings.Repeat(".", col.Width)
					}
				}
				
				// Pad to correct width considering visual width
				padding := col.Width - runewidth.StringWidth(cell)
				if padding > 0 {
					cell = cell + strings.Repeat(" ", padding)
				}
			} else {
				// Empty cell with proper padding
				cell = strings.Repeat(" ", col.Width)
			}
			
			rowContent += cell
		}
		// Apply appropriate style
		if t.focused && i == t.Selected {
			rowContent = t.styles.Selected.Render(rowContent)
		} else {
			rowContent = t.styles.Normal.Render(rowContent)
		}
		
		b.WriteString(rowContent)
	}
	
	return b.String()
}
