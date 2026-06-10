package style

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Column describes a single fixed-width table column.
type Column struct {
	Title string
	Width int
	Right bool // right-align the cell content
}

// HeaderRow renders the column titles in textDim, padded to their widths.
func HeaderRow(cols []Column) string {
	cells := make([]string, len(cols))
	for i, c := range cols {
		title := Truncate(c.Title, c.Width)
		if c.Right {
			cells[i] = Dim.Render(PadLeft(title, c.Width))
		} else {
			cells[i] = Dim.Render(PadRight(title, c.Width))
		}
	}
	return "  " + strings.Join(cells, "  ")
}

// Row renders a single data row. `values` holds already-styled strings and
// `widths` the visible (unstyled) widths used for padding; this lets callers
// color a cell while still aligning on its plain text length.
func Row(cols []Column, values []string, plain []string) string {
	cells := make([]string, len(cols))
	for i, c := range cols {
		styled := ""
		raw := ""
		if i < len(values) {
			styled = values[i]
		}
		if i < len(plain) {
			raw = plain[i]
		}
		cells[i] = padStyled(styled, raw, c.Width, c.Right)
	}
	return "  " + strings.Join(cells, "  ")
}

// padStyled pads a pre-styled string to width based on its plain-text length.
func padStyled(styled, plain string, width int, right bool) string {
	visible := lipgloss.Width(plain)
	if visible > width {
		// Re-truncate the plain text and drop styling to honor the column.
		return PadRight(Truncate(plain, width), width)
	}
	gap := width - visible
	if gap <= 0 {
		return styled
	}
	pad := strings.Repeat(" ", gap)
	if right {
		return pad + styled
	}
	return styled + pad
}
