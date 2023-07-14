package colors

type Color int

// This is taken from zerolog's repo and will be used to colorize log output
// Source: https://github.com/rs/zerolog/blob/4fff5db29c3403bc26dee9895e12a108aacc0203/console.go
const (
	// BLACK is the ANSI code for black
	BLACK Color = iota + 30
	// COLOR_RED is the ANSI code for red
	RED
	// GREEN is the ANSI code for green
	GREEN
	// YELLOW is the ANSI code for yellow
	YELLOW
	// BLUE is the ANSI code for blue
	BLUE
	// MAGENTA is the ANSI code for magenta
	MAGENTA
	// CYAN is the ANSI code for cyan
	CYAN
	// WHITE is the ANSI code for white
	WHITE
	// BOLD is the ANSI code for bold tet
	BOLD = 1
	// DARK_GRAY is the ANSI code for dark gray
	DARK_GRAY = 90
)

// This enum is to identify special unicode characters that will be used for pretty console output
const (
	// LEFT_ARROW is the unicode string for a left arrow glyph
	LEFT_ARROW = "\u21fe"
)
