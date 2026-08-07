package structure

import (
	"fmt"
	"strings"
)

// A minimal CIF 1.1 reader, covering the subset PDBx/mmCIF files actually use:
// data blocks, tag/value pairs, loop_ tables, single- and double-quoted
// strings, semicolon-delimited text fields, and the . / ? null markers.
// Save frames (save_) are skipped rather than parsed, since they appear only
// in dictionaries and never in coordinate files.

// cifBlock is one data_ block: categories mapped to their columns and rows.
// A tag/value pair is stored as a single-row table so that lookups do not need
// to distinguish the two forms.
type cifBlock struct {
	Name       string
	Categories map[string]*cifCategory
}

type cifCategory struct {
	Name    string
	Columns []string
	index   map[string]int
	Rows    [][]string
}

// Col returns the values of one column, or nil if the column is absent.
func (c *cifCategory) Col(name string) []string {
	if c == nil {
		return nil
	}
	i, ok := c.index[name]
	if !ok {
		return nil
	}
	out := make([]string, len(c.Rows))
	for r, row := range c.Rows {
		if i < len(row) {
			out[r] = row[i]
		}
	}
	return out
}

// First returns the first value of a column, or "" when absent or null.
func (c *cifCategory) First(name string) string {
	vals := c.Col(name)
	if len(vals) == 0 {
		return ""
	}
	return cifValue(vals[0])
}

func (b *cifBlock) Category(name string) *cifCategory {
	if b == nil {
		return nil
	}
	return b.Categories[name]
}

// cifValue normalises the CIF null markers to an empty string.
func cifValue(s string) string {
	if s == "." || s == "?" {
		return ""
	}
	return s
}

type cifToken struct {
	text  string
	quote byte // 0 for bare, else '\'', '"' or ';'
	line  int
}

func (t cifToken) isBare(s string) bool {
	return t.quote == 0 && strings.EqualFold(t.text, s)
}

func (t cifToken) isBarePrefix(s string) bool {
	return t.quote == 0 && len(t.text) >= len(s) && strings.EqualFold(t.text[:len(s)], s)
}

func cifTokenize(content string) ([]cifToken, error) {
	var toks []cifToken
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")

		// A semicolon in column 1 opens a multi-line text field that runs
		// until the next line whose first character is a semicolon.
		if strings.HasPrefix(line, ";") {
			var b strings.Builder
			b.WriteString(line[1:])
			start := i
			closed := false
			for i++; i < len(lines); i++ {
				cur := strings.TrimRight(lines[i], "\r")
				if strings.HasPrefix(cur, ";") {
					closed = true
					break
				}
				b.WriteString("\n")
				b.WriteString(cur)
			}
			if !closed {
				return nil, fmt.Errorf("unterminated text field starting on line %d", start+1)
			}
			toks = append(toks, cifToken{text: strings.TrimPrefix(b.String(), "\n"), quote: ';', line: start + 1})
			continue
		}

		rest := line
		col := 0
		for {
			trimmed := strings.TrimLeft(rest, " \t")
			col += len(rest) - len(trimmed)
			rest = trimmed
			if rest == "" || rest[0] == '#' {
				break
			}
			switch rest[0] {
			case '\'', '"':
				q := rest[0]
				// A quote only closes the string when followed by whitespace
				// or end of line, so apostrophes inside values are safe.
				end := -1
				for j := 1; j < len(rest); j++ {
					if rest[j] == q && (j+1 == len(rest) || rest[j+1] == ' ' || rest[j+1] == '\t') {
						end = j
						break
					}
				}
				if end < 0 {
					return nil, fmt.Errorf("unterminated quoted value on line %d", i+1)
				}
				toks = append(toks, cifToken{text: rest[1:end], quote: q, line: i + 1})
				rest = rest[end+1:]
			default:
				end := strings.IndexAny(rest, " \t")
				if end < 0 {
					end = len(rest)
				}
				toks = append(toks, cifToken{text: rest[:end], line: i + 1})
				rest = rest[end:]
			}
		}
	}
	return toks, nil
}

// cifParse builds the data blocks from a token stream.
func cifParse(content string) ([]*cifBlock, error) {
	toks, err := cifTokenize(content)
	if err != nil {
		return nil, err
	}

	var blocks []*cifBlock
	var cur *cifBlock
	newBlock := func(name string) {
		cur = &cifBlock{Name: name, Categories: make(map[string]*cifCategory)}
		blocks = append(blocks, cur)
	}

	put := func(tag string, values []string) {
		cat, item := splitCIFTag(tag)
		if cur == nil {
			newBlock("")
		}
		c, ok := cur.Categories[cat]
		if !ok {
			c = &cifCategory{Name: cat, index: make(map[string]int)}
			cur.Categories[cat] = c
		}
		if _, dup := c.index[item]; !dup {
			c.index[item] = len(c.Columns)
			c.Columns = append(c.Columns, item)
		}
		if len(c.Rows) == 0 {
			c.Rows = append(c.Rows, nil)
		}
		idx := c.index[item]
		for len(c.Rows[0]) <= idx {
			c.Rows[0] = append(c.Rows[0], "")
		}
		c.Rows[0][idx] = values[0]
	}

	for i := 0; i < len(toks); i++ {
		t := toks[i]
		switch {
		case t.isBarePrefix("data_"):
			newBlock(t.text[5:])

		case t.isBare("loop_"):
			var tags []string
			i++
			for ; i < len(toks) && toks[i].quote == 0 && strings.HasPrefix(toks[i].text, "_"); i++ {
				tags = append(tags, toks[i].text[1:])
			}
			if len(tags) == 0 {
				return nil, fmt.Errorf("loop_ with no tags on line %d", t.line)
			}
			cat, _ := splitCIFTag(tags[0])
			c := &cifCategory{Name: cat, index: make(map[string]int)}
			for _, tag := range tags {
				_, item := splitCIFTag(tag)
				if _, dup := c.index[item]; !dup {
					c.index[item] = len(c.Columns)
					c.Columns = append(c.Columns, item)
				}
			}
			var row []string
			for ; i < len(toks); i++ {
				v := toks[i]
				if v.quote == 0 && (strings.HasPrefix(v.text, "_") || v.isBare("loop_") ||
					v.isBarePrefix("data_") || v.isBarePrefix("save_") || v.isBare("stop_")) {
					break
				}
				row = append(row, v.text)
				if len(row) == len(tags) {
					c.Rows = append(c.Rows, row)
					row = nil
				}
			}
			i--
			if len(row) > 0 {
				return nil, fmt.Errorf("loop_ for category %q ended mid-row (%d of %d values)", cat, len(row), len(tags))
			}
			if cur == nil {
				newBlock("")
			}
			// Merge into any existing category of the same name rather than
			// silently discarding one of them.
			if prev, ok := cur.Categories[cat]; ok && len(prev.Rows) > 0 {
				c.Rows = append(prev.Rows, c.Rows...)
			}
			cur.Categories[cat] = c

		case t.quote == 0 && strings.HasPrefix(t.text, "_"):
			if i+1 >= len(toks) {
				return nil, fmt.Errorf("tag %s on line %d has no value", t.text, t.line)
			}
			put(t.text[1:], []string{toks[i+1].text})
			i++

		case t.isBarePrefix("save_"):
			// Skip save frames wholesale; they do not occur in coordinate files.
			for i++; i < len(toks) && !toks[i].isBarePrefix("save_"); i++ {
			}
		}
	}
	return blocks, nil
}

func splitCIFTag(tag string) (category, item string) {
	if i := strings.Index(tag, "."); i >= 0 {
		return strings.ToLower(tag[:i]), tag[i+1:]
	}
	return strings.ToLower(tag), ""
}
