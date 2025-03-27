package table

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

//------------------------------------------------------------------------------
// Column Definitions
//------------------------------------------------------------------------------

// Column represents a single table column.
type Column struct {
	Key      string // Unique identifier for the column
	Title    string // Display title for the column header
	Width    int    // Width of the column in characters
	Filtered bool   // Whether this column should be included in filtering
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

//------------------------------------------------------------------------------
// Row Definitions
//------------------------------------------------------------------------------

// RowData is a convenient alias for a map representing row data.
type RowData map[string]interface{}

// Row represents a single table row.
type Row struct {
	Data RowData // Map of column keys to cell values
}

// NewRow creates a new row from the given RowData.
func NewRow(data RowData) Row {
	return Row{Data: data}
}

//------------------------------------------------------------------------------
// Styling
//------------------------------------------------------------------------------

// Styles defines the visual appearance of different table elements
type Styles struct {
	Header   lipgloss.Style // Style for the table header
	Selected lipgloss.Style // Style for the selected row
	Normal   lipgloss.Style // Style for normal (unselected) rows
	Border   lipgloss.Style // Style for table borders
}

// DefaultStyles returns a set of default styles for the table
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

//------------------------------------------------------------------------------
// Table Definition
//------------------------------------------------------------------------------

// Table represents the table model.
type Table struct {
	Columns  []Column      // List of columns in the table
	Rows     []Row         // List of data rows
	PageSize int           // Number of rows to display per page
	Selected int           // Index of the currently selected row
	focused  bool          // Whether the table has focus
	filtered bool          // Whether filtering is enabled
	styles   Styles        // Visual styles for the table
	currentPage int        // Add this new field to track current page
	lastKey string        // Add this field to track the last key pressed for "gg" command
}

// New creates a new table instance with the provided columns.
func New(columns []Column) *Table {
	return &Table{
		Columns:    columns,
		Rows:       []Row{},
		PageSize:   10,
		Selected:   0,
		focused:    false,
		filtered:   false,
		currentPage: 0,    // Initialize current page
		lastKey:    "",
	}
}

//------------------------------------------------------------------------------
// Table Configuration Methods
//------------------------------------------------------------------------------

// WithPageSize sets the number of rows visible per page.
func (t *Table) WithPageSize(ps int) *Table {
	t.PageSize = ps
	return t
}

// WithFiltered sets whether the table has filtering enabled.
func (t *Table) WithFiltered(filtered bool) *Table {
	t.filtered = filtered
	return t
}

// WithFocused sets whether the table currently has focus.
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

//------------------------------------------------------------------------------
// Table Interaction Methods
//------------------------------------------------------------------------------

// Update handles key events and terminal resize
func (t *Table) Update(msg tea.Msg) (*Table, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		key := keyMsg.String()
		switch key {
		case "up", "k":
			if t.Selected > 0 {
				t.Selected--
				// Update page if selection moves above current page
				if t.Selected < t.currentPage*t.PageSize {
					t.currentPage--
				}
			}
		case "down", "j":
			if t.Selected < len(t.Rows)-1 {
				t.Selected++
				// Update page if selection moves below current page
				if t.Selected >= (t.currentPage+1)*t.PageSize {
					t.currentPage++
				}
			}
		case "h": // Page up
			t.currentPage = max(0, t.currentPage-1)
			t.Selected = max(0, t.currentPage*t.PageSize)
		case "l": // Page down
			maxPage := (len(t.Rows) - 1) / t.PageSize
			t.currentPage = min(maxPage, t.currentPage+1)
			t.Selected = min(len(t.Rows)-1, (t.currentPage+1)*t.PageSize-1)
		case "G": // Jump to bottom
			t.Selected = len(t.Rows) - 1
			t.currentPage = (t.Selected) / t.PageSize
		case "g": // Potential "gg" command
			if t.lastKey == "g" {
				t.Selected = 0
				t.currentPage = 0
				t.lastKey = ""
			} else {
				t.lastKey = "g"
			}
		default:
			t.lastKey = ""
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

//------------------------------------------------------------------------------
// Rendering
//------------------------------------------------------------------------------

// View renders the table as a string.
func (t *Table) View() string {
	var b strings.Builder
	
	// Create the header with proper width and alignment
	headerRow := ""
	for i, col := range t.Columns {
		if i > 0 {
			headerRow += "│" // Add column separator
		}
		// Adjust width consistently with other rows
		title := fmt.Sprintf("%-*s", col.Width, col.Title)
		if len(title) > col.Width {
			title = title[:col.Width-3] + "..." // Truncate with ellipsis if too long
		}
		headerRow += title
	}
	b.WriteString(t.styles.Header.Render(headerRow))
	
	// Add separator line with intersections
	b.WriteString("\n")
	separatorLine := ""
	for i := 0; i < len(t.Columns); i++ {
		if i > 0 {
			separatorLine += "┼" + strings.Repeat("─", t.Columns[i].Width) // Add intersection and horizontal line
		} else {
			separatorLine += strings.Repeat("─", t.Columns[i].Width) // Just horizontal line for first column
		}
	}
	b.WriteString(separatorLine)
	
	// Calculate visible rows for current page
	startIdx := t.currentPage * t.PageSize
	endIdx := min(startIdx+t.PageSize, len(t.Rows))
	visibleRows := t.Rows[startIdx:endIdx]
	
	// Render rows
	for i, row := range visibleRows {
		b.WriteString("\n")
		rowContent := ""
		for j, col := range t.Columns {
			if j > 0 {
				rowContent += "│" // Add column separator
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
						cell = truncated + "..." // Add ellipsis to indicate truncation
					} else {
						cell = strings.Repeat(".", col.Width) // For very narrow columns, just use dots
					}
				}
				
				// Pad to correct width considering visual width
				padding := col.Width - runewidth.StringWidth(cell)
				if padding > 0 {
					cell = cell + strings.Repeat(" ", padding) // Right-pad with spaces
				}
			} else {
				// Empty cell with proper padding
				cell = strings.Repeat(" ", col.Width)
			}
			
			rowContent += cell
		}
		// Adjust the selection highlighting to account for paging
		if t.focused && (startIdx+i) == t.Selected {
			rowContent = t.styles.Selected.Render(rowContent)
		} else {
			rowContent = t.styles.Normal.Render(rowContent)
		}
		
		b.WriteString(rowContent)
	}
	
	// Add page indicator
	totalPages := (len(t.Rows) + t.PageSize - 1) / t.PageSize
	if totalPages > 1 {
		b.WriteString(fmt.Sprintf("\nPage %d/%d", t.currentPage+1, totalPages))
	}
	
	return b.String()
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
