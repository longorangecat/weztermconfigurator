package ui

import (
	"sort"
	"strings"
	"unicode"

	"weztermconfigurator/internal/catalog"
)

// Fuzzy search over option names and their choice values.
//
// A term matches a target when its letters appear in the target in order
// (`wbo` → window_background_opacity, `fsz` → font_size). The best alignment
// is scored: letters right after a separator or the start count extra,
// consecutive letters count extra, gaps cost, and a plain substring gets a
// large bonus, so `scrollback` beats a scattered hit. Scattered hits with no
// structure score too low and are dropped, so the page only lists results the
// query really points at. Documentation text is deliberately not searched:
// fuzzy matching prose matches nearly everything.

const (
	fuzzyChar       = 16 // every matched letter
	fuzzyConsec     = 24 // letter directly after the previous match
	fuzzyBoundary   = 20 // letter at the start of a word (start or after _ - . space)
	fuzzyGapCost    = 3  // per skipped letter between two matches...
	fuzzyGapCap     = 15 // ...up to this much per gap
	fuzzyLeadCap    = 10 // leading skipped letters cost 1 each up to this
	fuzzyLoose      = 12 // extra cost of a letter that is neither consecutive nor at a word start
	fuzzySubstring  = 60 // the term occurs as a contiguous run
	fuzzyPrefix     = 40 // ...and starts the target or a word
	fuzzyChoicePen  = 25 // a choice value ranks below the same hit on the name
	fuzzyMinPerChar = 20 // average score per term letter needed to count as a match
	searchCap       = 60 // rows shown for a search; more would be noise (and slow to build)
	fuzzyNegInf     = -1 << 30
)

func isSep(r rune) bool { return r == '_' || r == '-' || r == '.' || r == ' ' || r == '/' }

// fuzzyTerm scores one lower-case term against one lower-case target.
func fuzzyTerm(term, text []rune) (int, bool) {
	m, n := len(term), len(text)
	if m == 0 || m > n {
		return 0, false
	}
	boundary := func(i int) bool { return i == 0 || isSep(text[i-1]) }

	// one letter would match nearly everything: it must start a word
	if m == 1 {
		for i := range text {
			if text[i] == term[0] && boundary(i) {
				return fuzzyChar + fuzzyBoundary, true
			}
		}
		return 0, false
	}

	sub := indexRunes(text, term)

	// dp[i]: best score with term[:k+1] aligned and its last letter at text[i]
	prev := make([]int, n)
	cur := make([]int, n)
	for i := range n {
		prev[i] = fuzzyNegInf
		// a scattered match must start on a word; mid-word starts are only for exact substrings
		if text[i] == term[0] && (boundary(i) || sub >= 0) {
			s := fuzzyChar - min(i, fuzzyLeadCap)
			if boundary(i) {
				s += fuzzyBoundary
			}
			prev[i] = s
		}
	}
	for k := 1; k < m; k++ {
		for i := range n {
			cur[i] = fuzzyNegInf
			if text[i] != term[k] {
				continue
			}
			best := fuzzyNegInf
			bnd := boundary(i)
			for j := range i {
				if prev[j] == fuzzyNegInf {
					continue
				}
				s := prev[j]
				if i == j+1 {
					s += fuzzyConsec
				} else {
					s -= min(fuzzyGapCost*(i-j-1), fuzzyGapCap)
					if !bnd {
						s -= fuzzyLoose
					}
				}
				if s > best {
					best = s
				}
			}
			if best == fuzzyNegInf {
				continue
			}
			best += fuzzyChar
			if bnd {
				best += fuzzyBoundary
			}
			cur[i] = best
		}
		prev, cur = cur, prev
	}
	score := fuzzyNegInf
	for _, s := range prev {
		score = max(score, s)
	}
	if score == fuzzyNegInf {
		return 0, false
	}

	if sub >= 0 {
		score += fuzzySubstring
		if boundary(sub) {
			score += fuzzyPrefix
		}
	}
	if score < fuzzyMinPerChar*m {
		return 0, false
	}
	return score, true
}

func indexRunes(text, term []rune) int {
	for i := 0; i+len(term) <= len(text); i++ {
		match := true
		for j := range term {
			if text[i+j] != term[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func lowerRunes(s string) []rune {
	r := []rune(s)
	for i, c := range r {
		r[i] = unicode.ToLower(c)
	}
	return r
}

// optionSearchScore scores an option for a query of whitespace-separated
// terms. Every term must match, each against the option name or one of its
// top-level choice values (whichever fits it best); ok is false otherwise.
func optionSearchScore(o *catalog.Option, q string) (int, bool) {
	terms := strings.Fields(strings.ToLower(q))
	if len(terms) == 0 {
		return 0, false
	}
	name := lowerRunes(o.Name)
	choices := make([][]rune, len(o.Enum))
	for i, e := range o.Enum {
		choices[i] = lowerRunes(e)
	}
	total := 0
	for _, t := range terms {
		tr := []rune(t)
		best, found := fuzzyTerm(tr, name)
		for _, c := range choices {
			if s, ok := fuzzyTerm(tr, c); ok && (!found || s-fuzzyChoicePen > best) {
				best, found = s-fuzzyChoicePen, true
			}
		}
		if !found {
			return 0, false
		}
		total += best
	}
	return total, true
}

// rankOptions returns the options matching q, best first, ties in catalog order.
func rankOptions(opts []*catalog.Option, q string) []*catalog.Option {
	type hit struct {
		o     *catalog.Option
		score int
	}
	var hits []hit
	for _, o := range opts {
		if s, ok := optionSearchScore(o, q); ok {
			hits = append(hits, hit{o, s})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	// Weak scattered hits next to a strong one are noise (`opac` would also list
	// mux_output_parser_coalesce…): keep what is at least half as good as the best.
	for i, h := range hits {
		if h.score*2 < hits[0].score {
			hits = hits[:i]
			break
		}
	}
	out := make([]*catalog.Option, len(hits))
	for i, h := range hits {
		out[i] = h.o
	}
	return out
}
