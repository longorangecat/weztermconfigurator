package lint

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func fmtDiff(lines []DiffLine) string {
	var b strings.Builder
	for _, l := range lines {
		fmt.Fprintf(&b, "%c%s|old=%d|new=%d\n", l.Op, l.Text, l.OldNo, l.NewNo)
	}
	return b.String()
}

func TestDiff(t *testing.T) {
	tests := []struct {
		name     string
		old, new string
		want     string
	}{
		{
			name: "both empty",
			want: "",
		},
		{
			name: "empty to content",
			new:  "a\nb",
			want: "+a|old=0|new=1\n+b|old=0|new=2\n",
		},
		{
			name: "content to empty",
			old:  "a\nb",
			want: "-a|old=1|new=0\n-b|old=2|new=0\n",
		},
		{
			name: "identical",
			old:  "a\nb\nc",
			new:  "a\nb\nc",
			want: " a|old=1|new=1\n b|old=2|new=2\n c|old=3|new=3\n",
		},
		{
			name: "trailing newline is not a change",
			old:  "a\nb",
			new:  "a\nb\n",
			want: " a|old=1|new=1\n b|old=2|new=2\n",
		},
		{
			name: "crlf is not a change and is stripped",
			old:  "a\r\nb\r\n",
			new:  "a\nb",
			want: " a|old=1|new=1\n b|old=2|new=2\n",
		},
		{
			name: "modified middle keeps line numbers",
			old:  "a\nb\nc",
			new:  "a\nB\nc",
			want: " a|old=1|new=1\n-b|old=2|new=0\n+B|old=0|new=2\n c|old=3|new=3\n",
		},
		{
			name: "added line",
			old:  "a\nc",
			new:  "a\nb\nc",
			want: " a|old=1|new=1\n+b|old=0|new=2\n c|old=2|new=3\n",
		},
		{
			name: "removed line",
			old:  "a\nb\nc",
			new:  "a\nc",
			want: " a|old=1|new=1\n-b|old=2|new=0\n c|old=3|new=2\n",
		},
		{
			name: "blank lines count as lines",
			old:  "a\n\nb",
			new:  "a\nb",
			want: " a|old=1|new=1\n-|old=2|new=0\n b|old=3|new=2\n",
		},
		{
			// A moved block is a removal plus an addition; the LCS keeps the
			// shared tail aligned.
			name: "reordered",
			old:  "a\nb\nc\nd",
			new:  "b\nc\nd\na",
			want: "-a|old=1|new=0\n b|old=2|new=1\n c|old=3|new=2\n d|old=4|new=3\n+a|old=0|new=4\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := fmtDiff(Diff(tc.old, tc.new)); got != tc.want {
				t.Errorf("Diff(%q, %q):\n%s\nwant:\n%s", tc.old, tc.new, got, tc.want)
			}
		})
	}
}

func TestCollapse(t *testing.T) {
	lines := make([]DiffLine, 0, 10)
	for i := range 10 {
		op := byte(' ')
		if i == 1 || i == 8 {
			op = '+'
		}
		lines = append(lines, DiffLine{Op: op, Text: fmt.Sprint(i)})
	}
	tests := []struct {
		name    string
		lines   []DiffLine
		context int
		want    string
	}{
		{name: "empty", lines: nil, context: 2, want: ""},
		{
			name:    "no changes folds to one marker",
			lines:   []DiffLine{{Op: ' ', Text: "a"}, {Op: ' ', Text: "b"}, {Op: ' ', Text: "c"}},
			context: 1,
			want:    "~… 3 unchanged lines",
		},
		{
			name:    "zero context keeps only changes",
			lines:   lines,
			context: 0,
			want:    "~… 1 unchanged lines\n+1\n~… 6 unchanged lines\n+8\n~… 1 unchanged lines",
		},
		{
			name:    "context of one",
			lines:   lines,
			context: 1,
			want:    " 0\n+1\n 2\n~… 4 unchanged lines\n 7\n+8\n 9",
		},
		{
			name:    "context covers everything",
			lines:   lines,
			context: 5,
			want:    " 0\n+1\n 2\n 3\n 4\n 5\n 6\n 7\n+8\n 9",
		},
		{
			name:    "negative context is zero",
			lines:   lines,
			context: -1,
			want:    "~… 1 unchanged lines\n+1\n~… 6 unchanged lines\n+8\n~… 1 unchanged lines",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, l := range Collapse(tc.lines, tc.context) {
				got = append(got, string(l.Op)+l.Text)
			}
			if strings.Join(got, "\n") != tc.want {
				t.Errorf("Collapse(_, %d) =\n%s\nwant:\n%s", tc.context, strings.Join(got, "\n"), tc.want)
			}
		})
	}
}

func TestStats(t *testing.T) {
	lines := Diff("a\nb\nc\nd\ne", "a\nB\nc\nd\nf\ng")
	added, removed := Stats(lines)
	if added != 3 || removed != 2 {
		t.Errorf("Stats = %d, %d; want 3, 2 (%s)", added, removed, fmtDiff(lines))
	}
	added, removed = Stats(Collapse(lines, 0))
	if added != 3 || removed != 2 {
		t.Errorf("Stats after Collapse = %d, %d; want 3, 2", added, removed)
	}
}

// TestDiffLarge guards the memory-blow-up ceiling at the size a real config can
// reach.
func TestDiffLarge(t *testing.T) {
	const n = 5000
	var b strings.Builder
	for i := range n {
		fmt.Fprintf(&b, "config.option_%d = %d\n", i, i)
	}
	oldText := b.String()

	edit := func(idxs []int) string {
		out := oldText
		for _, i := range idxs {
			out = strings.Replace(out,
				fmt.Sprintf("config.option_%d = %d\n", i, i),
				fmt.Sprintf("config.option_%d = %d\n", i, i+1), 1)
		}
		return out
	}
	clustered := make([]int, 50)
	for i := range clustered {
		clustered[i] = 1000 + i
	}
	scattered := make([]int, 50)
	seed := uint64(12345)
	for i := range scattered {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		scattered[i] = int(seed % n)
	}

	t.Run("localized edit is minimal", func(t *testing.T) {
		start := time.Now()
		lines := Diff(oldText, edit(clustered))
		if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
			t.Errorf("Diff of %d lines took %v; want well under a second", n, elapsed)
		}
		if added, removed := Stats(lines); added != 50 || removed != 50 {
			t.Errorf("Stats = %d, %d; want 50, 50", added, removed)
		}
		if len(lines) != n+50 {
			t.Errorf("len(lines) = %d; want %d", len(lines), n+50)
		}
	})

	t.Run("edits across the whole file stay fast", func(t *testing.T) {
		// The changed block is now the whole file, past the LCS cell budget,
		// so this exercises the block-replace fallback.
		start := time.Now()
		lines := Diff(oldText, edit(scattered))
		if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
			t.Errorf("Diff of %d lines took %v; want well under a second", n, elapsed)
		}
		if added, removed := Stats(lines); added < 50 || removed < 50 {
			t.Errorf("Stats = %d, %d; want at least 50, 50", added, removed)
		}
	})
}
