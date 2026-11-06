package fsx

import "strings"

// DiffRow is one row of a line diff: equal (' '), removed ('-') or added
// ('+'), with the line numbers on each side, 1-based, 0 for none.
type DiffRow struct {
	Kind byte
	ANo  int
	BNo  int
	A    string
	B    string
}

// DiffLines compares two texts line by line. Very large texts fall back to a
// coarse diff that marks the changed middle as removed and added.
func DiffLines(aText, bText string) []DiffRow {
	a := splitLines(aText)
	b := splitLines(bText)
	if len(a)+len(b) > 6000 {
		return coarseDiff(a, b)
	}
	return myersDiff(a, b)
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

type diffOp struct {
	kind byte
	a, b int // indices, -1 for none
}

func myersDiff(a, b []string) []DiffRow {
	n, m := len(a), len(b)
	max := n + m
	if max == 0 {
		return nil
	}
	v := make([]int, 2*max+2)
	offset := max + 1
	trace := make([][]int, 0, max+1)
	var ops []diffOp
	found := false
	for d := 0; d <= max; d++ {
		snapshot := make([]int, len(v))
		copy(snapshot, v)
		trace = append(trace, snapshot)
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[offset+k-1] < v[offset+k+1]) {
				x = v[offset+k+1]
			} else {
				x = v[offset+k-1] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[offset+k] = x
			if x >= n && y >= m {
				found = true
				break
			}
		}
		if found {
			// Backtrack from (n, m).
			x, y := n, m
			for d := len(trace) - 1; d >= 0; d-- {
				vv := trace[d]
				k := x - y
				var prevK int
				if k == -d || (k != d && vv[offset+k-1] < vv[offset+k+1]) {
					prevK = k + 1
				} else {
					prevK = k - 1
				}
				prevX := vv[offset+prevK]
				prevY := prevX - prevK
				for x > prevX && y > prevY {
					ops = append(ops, diffOp{' ', x - 1, y - 1})
					x--
					y--
				}
				if d > 0 {
					if x == prevX {
						ops = append(ops, diffOp{'+', -1, y - 1})
					} else {
						ops = append(ops, diffOp{'-', x - 1, -1})
					}
				}
				x, y = prevX, prevY
			}
			break
		}
	}
	// ops is in reverse order.
	rows := make([]DiffRow, 0, len(ops))
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		switch op.kind {
		case ' ':
			rows = append(rows, DiffRow{' ', op.a + 1, op.b + 1, a[op.a], b[op.b]})
		case '-':
			rows = append(rows, DiffRow{'-', op.a + 1, 0, a[op.a], ""})
		case '+':
			rows = append(rows, DiffRow{'+', 0, op.b + 1, "", b[op.b]})
		}
	}
	return rows
}

func coarseDiff(a, b []string) []DiffRow {
	var rows []DiffRow
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		rows = append(rows, DiffRow{' ', pre + 1, pre + 1, a[pre], b[pre]})
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	for i := pre; i < len(a)-suf; i++ {
		rows = append(rows, DiffRow{'-', i + 1, 0, a[i], ""})
	}
	for i := pre; i < len(b)-suf; i++ {
		rows = append(rows, DiffRow{'+', 0, i + 1, "", b[i]})
	}
	for i := len(a) - suf; i < len(a); i++ {
		rows = append(rows, DiffRow{' ', i + 1, len(b) - suf + (i - (len(a) - suf)) + 1, a[i], b[len(b)-suf+(i-(len(a)-suf))]})
	}
	return rows
}

// DiffStats counts the equal, removed and added rows.
func DiffStats(rows []DiffRow) (equal, removed, added int) {
	for _, r := range rows {
		switch r.Kind {
		case ' ':
			equal++
		case '-':
			removed++
		case '+':
			added++
		}
	}
	return
}
