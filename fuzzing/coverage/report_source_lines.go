package coverage

import (
	"bytes"
	"html"
)

// coverageSourceLine indicates whether the
type coverageSourceLine struct {
	IsActive  bool
	Start     int
	End       int
	Contents  []byte
	IsCovered bool
}

func (sl *coverageSourceLine) ContentsHTML() string {
	return html.EscapeString(string(sl.Contents))
}

// splitSourceCode splits the provided source code into coverageSourceLine objects.
// Returns the coverageSourceLine objects.
func splitSourceCode(sourceCode []byte) []*coverageSourceLine {
	// Create our lines and a variable to track where our current line start offset is.
	var lines []*coverageSourceLine
	var lineStart int

	// Split the source code on new line characters
	sourceCodeLinesBytes := bytes.Split(sourceCode, []byte("\n"))

	// For each source code line, initialize a struct that defines its start/end offsets, set its contents.
	for i := 0; i < len(sourceCodeLinesBytes); i++ {
		lineEnd := lineStart + len(sourceCodeLinesBytes[i]) + 1
		lines = append(lines, &coverageSourceLine{
			IsActive:  false,
			Start:     lineStart,
			End:       lineEnd,
			Contents:  sourceCodeLinesBytes[i],
			IsCovered: false,
		})
		lineStart = lineEnd
	}

	// Return the resulting lines
	return lines
}
