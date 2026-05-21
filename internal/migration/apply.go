package migration

import (
	"fmt"
	"go/token"
	"sort"
)

type offsetEdit struct {
	edit       Edit
	start, end int
}

func Apply(src []byte, fset *token.FileSet, edits []Edit) ([]byte, error) {
	converted := make([]offsetEdit, 0, len(edits))
	for _, edit := range edits {
		start := fset.Position(edit.Start).Offset
		end := fset.Position(edit.End).Offset
		if start < 0 || end < start || end > len(src) {
			return nil, fmt.Errorf("invalid edit range for %s at %s:%d:%d", edit.Category, edit.Position.Filename, edit.Position.Line, edit.Position.Column)
		}
		converted = append(converted, offsetEdit{edit: edit, start: start, end: end})
	}
	sort.Slice(converted, func(i, j int) bool {
		if converted[i].start == converted[j].start {
			return converted[i].end > converted[j].end
		}
		return converted[i].start < converted[j].start
	})
	for i := 1; i < len(converted); i++ {
		if converted[i].start < converted[i-1].end {
			return nil, fmt.Errorf("conflicting edits: %s overlaps %s", converted[i-1].edit.Category, converted[i].edit.Category)
		}
	}
	out := append([]byte(nil), src...)
	for i := len(converted) - 1; i >= 0; i-- {
		edit := converted[i]
		next := make([]byte, 0, len(out)-(edit.end-edit.start)+len(edit.edit.NewText))
		next = append(next, out[:edit.start]...)
		next = append(next, edit.edit.NewText...)
		next = append(next, out[edit.end:]...)
		out = next
	}
	return out, nil
}
