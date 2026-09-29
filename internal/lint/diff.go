// Package lint holds small static checks over the generated config: a line
// diff for the pre-save review dialog and key binding conflict detection.
package lint

import (
	"strconv"
	"strings"
)

// DiffLine is one line of a line-based diff. Op is ' ' (unchanged), '+'
// (added), '-' (removed) or '~' (a fold marker produced by Collapse). OldNo
// and NewNo are 1-based line numbers in the respective text, 0 where the line
// does not exist on that side.
type DiffLine struct {
	Op    byte
	Text  string
	OldNo int
	NewNo int
}

// maxDPCells caps the LCS table at 4M int32 cells (16 MiB). Config files are
// far below this once the common prefix and suffix are trimmed; a pathological
// whole-file rewrite falls back to a block replace, which is valid but not
// minimal.
// ponytail: block replace above 4M cells; use a patience/anchor diff if real
// configs ever get there.
const maxDPCells = 1 << 22

// Diff compares two texts line by line. A missing trailing newline is not a
// difference, and a trailing \r (CRLF files) is stripped from both the
// comparison and Text. Empty text has no lines.
func Diff(oldText, newText string) []DiffLine {
	oldLines, newLines := splitLines(oldText), splitLines(newText)

	// Trim the common prefix and suffix: for a config edited in place this
	// leaves only the changed block, which keeps the LCS table tiny.
	pre := 0
	for pre < len(oldLines) && pre < len(newLines) && oldLines[pre] == newLines[pre] {
		pre++
	}
	suf := 0
	for suf < len(oldLines)-pre && suf < len(newLines)-pre &&
		oldLines[len(oldLines)-1-suf] == newLines[len(newLines)-1-suf] {
		suf++
	}

	out := make([]DiffLine, 0, len(oldLines)+len(newLines))
	for i := range pre {
		out = append(out, DiffLine{Op: ' ', Text: oldLines[i], OldNo: i + 1, NewNo: i + 1})
	}
	diffBlock(oldLines[pre:len(oldLines)-suf], newLines[pre:len(newLines)-suf], pre, pre, &out)
	for i := len(oldLines) - suf; i < len(oldLines); i++ {
		out = append(out, DiffLine{Op: ' ', Text: oldLines[i], OldNo: i + 1, NewNo: i + 1 + len(newLines) - len(oldLines)})
	}
	return out
}

// diffBlock appends the diff of a and b, where oldBase/newBase are the 0-based
// offsets of a[0]/b[0] within their full text.
func diffBlock(a, b []string, oldBase, newBase int, out *[]DiffLine) {
	p, q := len(a), len(b)
	if p == 0 && q == 0 {
		return
	}
	if p == 0 || q == 0 || p*q > maxDPCells {
		for i, s := range a {
			*out = append(*out, DiffLine{Op: '-', Text: s, OldNo: oldBase + i + 1})
		}
		for j, s := range b {
			*out = append(*out, DiffLine{Op: '+', Text: s, NewNo: newBase + j + 1})
		}
		return
	}

	// dp[i*(q+1)+j] = LCS length of a[i:] and b[j:].
	stride := q + 1
	dp := make([]int32, (p+1)*stride)
	for i := p - 1; i >= 0; i-- {
		for j := q - 1; j >= 0; j-- {
			n := int32(0)
			if a[i] == b[j] {
				n = dp[(i+1)*stride+j+1] + 1
			} else if dp[(i+1)*stride+j] >= dp[i*stride+j+1] {
				n = dp[(i+1)*stride+j]
			} else {
				n = dp[i*stride+j+1]
			}
			dp[i*stride+j] = n
		}
	}
	i, j := 0, 0
	for i < p && j < q {
		switch {
		case a[i] == b[j]:
			*out = append(*out, DiffLine{Op: ' ', Text: a[i], OldNo: oldBase + i + 1, NewNo: newBase + j + 1})
			i++
			j++
		case dp[(i+1)*stride+j] >= dp[i*stride+j+1]:
			*out = append(*out, DiffLine{Op: '-', Text: a[i], OldNo: oldBase + i + 1})
			i++
		default:
			*out = append(*out, DiffLine{Op: '+', Text: b[j], NewNo: newBase + j + 1})
			j++
		}
	}
	for ; i < p; i++ {
		*out = append(*out, DiffLine{Op: '-', Text: a[i], OldNo: oldBase + i + 1})
	}
	for ; j < q; j++ {
		*out = append(*out, DiffLine{Op: '+', Text: b[j], NewNo: newBase + j + 1})
	}
}

// Collapse keeps changed lines plus context unchanged lines around them and
// folds each run of unchanged lines that is not within context of a change
// into a single '~' marker. Input that is entirely unchanged collapses to one
// marker (not the full listing), so a review dialog can show a one-line
// "nothing changed" summary; empty input stays empty.
func Collapse(lines []DiffLine, context int) []DiffLine {
	if len(lines) == 0 {
		return nil
	}
	if context < 0 {
		context = 0
	}

	keep := make([]bool, len(lines))
	changed := false
	for i, l := range lines {
		if l.Op == '+' || l.Op == '-' {
			changed = true
			for j := max(0, i-context); j < min(len(lines), i+context+1); j++ {
				keep[j] = true
			}
		}
	}
	if !changed {
		return []DiffLine{{Op: '~', Text: foldText(len(lines))}}
	}

	var out []DiffLine
	for i := 0; i < len(lines); {
		if !keep[i] {
			j := i
			for j < len(lines) && !keep[j] {
				j++
			}
			out = append(out, DiffLine{Op: '~', Text: foldText(j - i)})
			i = j
			continue
		}
		out = append(out, lines[i])
		i++
	}
	return out
}

func foldText(n int) string {
	return "… " + strconv.Itoa(n) + " unchanged lines"
}

// Stats counts added and removed lines; fold markers and unchanged lines are
// ignored.
func Stats(lines []DiffLine) (added, removed int) {
	for _, l := range lines {
		switch l.Op {
		case '+':
			added++
		case '-':
			removed++
		}
	}
	return added, removed
}

// splitLines splits text into lines without terminators. CRLF is normalized and
// a trailing newline does not produce an extra empty line.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	text = strings.TrimSuffix(text, "\n")
	out := strings.Split(text, "\n")
	for i, s := range out {
		out[i] = strings.TrimSuffix(s, "\r")
	}
	return out
}
