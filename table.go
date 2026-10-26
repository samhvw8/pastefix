package main

import (
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/mattn/go-runewidth"
)

// Table is a parsed text table. Header is nil when the table has none.
type Table struct {
	Header []string
	Rows   [][]string
}

const (
	boxRuleChars  = "=─━═┼┬┴├┤┌┐└┘╞╡╪╤╧╒╕╘╛╔╗╚╝╠╣╦╩╬┏┓┗┛┣┫┳┻╋┡┩╇+"
	heavyRuleChar = "═━="
	mdRuleChars   = "-:|│┃║ "
)

var errNoTable = errors.New("no table rows ('|' or '│') found in input")

// ruleKind returns "box" or "md" for a rule or separator line, "" for a data line.
func ruleKind(line string) string {
	s := strings.TrimSpace(line)
	if s == "" {
		return ""
	}
	hasBox := false
	for _, r := range s {
		switch {
		case strings.ContainsRune(boxRuleChars, r):
			hasBox = true
		case strings.ContainsRune(mdRuleChars, r):
		default:
			return ""
		}
	}
	if hasBox {
		return "box"
	}
	if strings.Contains(s, "-") {
		return "md"
	}
	return ""
}

func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	if strings.HasSuffix(line, "|") && !strings.HasSuffix(line, `\|`) {
		line = line[:len(line)-1]
	}
	// Split on pipes; `\|` is a literal pipe inside a cell.
	var parts []string
	var cur strings.Builder
	for i := 0; i < len(line); i++ {
		switch {
		case line[i] == '\\' && i+1 < len(line) && line[i+1] == '|':
			cur.WriteByte('|')
			i++
		case line[i] == '|':
			parts = append(parts, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(line[i])
		}
	}
	return append(parts, strings.TrimSpace(cur.String()))
}

// Parse reads a Markdown pipe table or a box-drawing table (│ ├─┼─┤, +---+).
// In a box table the lines between two rules are one row, so cells wrapped
// over several lines are joined again.
func Parse(text string) (Table, error) {
	repl := strings.NewReplacer("│", "|", "┃", "|", "║", "|")
	lines := strings.Split(repl.Replace(strings.ReplaceAll(text, "\r\n", "\n")), "\n")
	boxed := false
	for _, l := range lines {
		if ruleKind(l) == "box" {
			boxed = true
			break
		}
	}

	var rows [][]string
	rules := map[int]bool{} // rule position (rows before it) -> heavy
	var group [][]string
	flush := func() {
		if len(group) == 0 {
			return
		}
		ncol := 0
		for _, g := range group {
			ncol = max(ncol, len(g))
		}
		row := make([]string, ncol)
		for i := range row {
			var bits []string
			for _, g := range group {
				if i < len(g) && g[i] != "" {
					bits = append(bits, g[i])
				}
			}
			row[i] = strings.Join(bits, " ")
		}
		rows = append(rows, row)
		group = nil
	}

	for _, raw := range lines {
		if ruleKind(raw) != "" {
			flush()
			rules[len(rows)] = rules[len(rows)] || strings.ContainsAny(raw, heavyRuleChar)
			continue
		}
		if !strings.Contains(raw, "|") {
			flush()
			continue
		}
		cells := splitRow(raw)
		empty := true
		for _, c := range cells {
			if c != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		group = append(group, cells)
		if !boxed {
			flush()
		}
	}
	flush()
	if len(rows) == 0 {
		return Table{}, errNoTable
	}

	ncol := 0
	for _, r := range rows {
		ncol = max(ncol, len(r))
	}
	for i, r := range rows {
		for len(r) < ncol {
			r = append(r, "")
		}
		rows[i] = r
	}

	// A header has its own rule after row 1 (or a heavy ═/━ one) and no other
	// rules between rows. Rules after every row mean there is no header.
	inner := 0
	onlyAfterFirst := true
	for p := range rules {
		if p > 0 && p < len(rows) {
			inner++
			if p != 1 {
				onlyAfterFirst = false
			}
		}
	}
	t := Table{Rows: rows}
	if len(rows) > 1 && ((inner == 1 && onlyAfterFirst) || rules[1]) {
		t.Header, t.Rows = rows[0], rows[1:]
	}
	return t, nil
}

// SetHeader names the columns, or with names == nil turns the header back into a row.
func (t *Table) SetHeader(names []string) error {
	if names == nil {
		if t.Header != nil {
			t.Rows = append([][]string{t.Header}, t.Rows...)
			t.Header = nil
		}
		return nil
	}
	if len(names) != len(t.Rows[0]) {
		return fmt.Errorf("--header has %d names but the table has %d columns", len(names), len(t.Rows[0]))
	}
	t.Header = names
	return nil
}

func width(s string) int { return runewidth.StringWidth(s) }

func pad(s string, w int) string { return s + strings.Repeat(" ", max(0, w-width(s))) }

// wrap breaks s into lines of at most limit display columns, at spaces only
// (a long word stays whole). limit 0 means no wrapping.
func wrap(s string, limit int) []string {
	if limit <= 0 {
		return []string{s}
	}
	var lines []string
	cur := ""
	for _, w := range strings.Fields(s) {
		switch {
		case cur == "":
			cur = w
		case width(cur)+1+width(w) <= limit:
			cur += " " + w
		default:
			lines = append(lines, cur)
			cur = w
		}
	}
	if cur != "" || len(lines) == 0 {
		lines = append(lines, cur)
	}
	return lines
}

// Box renders a box-drawing table. The rule under a header is heavy (╞═╪═╡),
// so Parse reads the header back.
func Box(t Table, limit int) string {
	all := t.Rows
	if t.Header != nil {
		all = append([][]string{t.Header}, t.Rows...)
	}
	wrapped := make([][][]string, len(all))
	widths := make([]int, len(all[0]))
	for r, row := range all {
		wrapped[r] = make([][]string, len(row))
		for c, cell := range row {
			wrapped[r][c] = wrap(cell, limit)
			for _, l := range wrapped[r][c] {
				widths[c] = max(widths[c], width(l))
			}
		}
	}
	rule := func(l, m, r, h string) string {
		parts := make([]string, len(widths))
		for i, w := range widths {
			parts[i] = strings.Repeat(h, w+2)
		}
		return l + strings.Join(parts, m) + r
	}

	out := []string{rule("┌", "┬", "┐", "─")}
	for n, cells := range wrapped {
		height := 0
		for _, c := range cells {
			height = max(height, len(c))
		}
		for k := 0; k < height; k++ {
			var b strings.Builder
			b.WriteString("│")
			for i, c := range cells {
				line := ""
				if k < len(c) {
					line = c[k]
				}
				b.WriteString(" " + pad(line, widths[i]) + " │")
			}
			out = append(out, b.String())
		}
		if n < len(wrapped)-1 {
			if t.Header != nil && n == 0 {
				out = append(out, rule("╞", "╪", "╡", "═"))
			} else {
				out = append(out, rule("├", "┼", "┤", "─"))
			}
		}
	}
	out = append(out, rule("└", "┴", "┘", "─"))
	return strings.Join(out, "\n")
}

// Markdown renders an aligned pipe table, naming columns "Column N" when there is no header.
func Markdown(t Table) string {
	head := t.Header
	if head == nil {
		head = make([]string, len(t.Rows[0]))
		for i := range head {
			head[i] = fmt.Sprintf("Column %d", i+1)
		}
	}
	esc := func(s string) string { return strings.ReplaceAll(s, "|", `\|`) }
	widths := make([]int, len(head))
	for _, r := range append([][]string{head}, t.Rows...) {
		for i, c := range r {
			widths[i] = max(widths[i], width(esc(c)), 3)
		}
	}
	line := func(r []string) string {
		cells := make([]string, len(r))
		for i, c := range r {
			cells[i] = pad(esc(c), widths[i])
		}
		return "| " + strings.Join(cells, " | ") + " |"
	}
	sep := make([]string, len(widths))
	for i, w := range widths {
		sep[i] = strings.Repeat("-", w)
	}
	out := []string{line(head), "| " + strings.Join(sep, " | ") + " |"}
	for _, r := range t.Rows {
		out = append(out, line(r))
	}
	return strings.Join(out, "\n")
}

// List renders one block per row: the first cell as a title, then a bullet per other cell.
func List(t Table, bold string) string {
	blocks := make([]string, 0, len(t.Rows))
	for _, r := range t.Rows {
		lines := []string{bold + r[0] + bold}
		for i, c := range r[1:] {
			if c == "" {
				continue
			}
			if t.Header != nil {
				lines = append(lines, "• "+t.Header[i+1]+": "+c)
			} else {
				lines = append(lines, "• "+c)
			}
		}
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	return strings.Join(blocks, "\n\n")
}

// HTML renders a bordered table for rich-text paste.
func HTML(t Table) string {
	cell := func(tag, c string) string {
		return fmt.Sprintf(`<%s style="border:1px solid #999;padding:4px 8px;text-align:left;vertical-align:top">%s</%s>`, tag, html.EscapeString(c), tag)
	}
	var b strings.Builder
	b.WriteString(`<html><head><meta charset="utf-8"></head><body><table style="border-collapse:collapse">`)
	row := func(tag string, r []string) {
		b.WriteString("<tr>")
		for _, c := range r {
			b.WriteString(cell(tag, c))
		}
		b.WriteString("</tr>")
	}
	if t.Header != nil {
		row("th", t.Header)
	}
	for _, r := range t.Rows {
		row("td", r)
	}
	b.WriteString("</table></body></html>")
	return b.String()
}
