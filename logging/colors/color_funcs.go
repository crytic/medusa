package colors

import "fmt"

// ColorFunc is an alias type for a coloring function that accepts anything and returns a colorized string
type ColorFunc = func(s any) string

// Reset is a ColorFunc that simply returns the input as a string. It is basically a no-op and is used for resetting the
// color context during complex logging operations.
func Reset(s any) string {
	return fmt.Sprintf("%v", s)
}

// Black is a ColorFunc that returns a black-colorized string of the provided input
func Black(s any) string {
	return Colorize(s, BLACK)
}

// BlackBold is a ColorFunc that returns a black-bold-colorized string of the provided input
func BlackBold(s any) string {
	return Colorize(Colorize(s, BLACK), BOLD)
}

// Red is a ColorFunc that returns a red-colorized string of the provided input
func Red(s any) string {
	return Colorize(s, RED)
}

// RedBold is a ColorFunc that returns a red-bold-colorized string of the provided input
func RedBold(s any) string {
	return Colorize(Colorize(s, RED), BOLD)
}

// Green is a ColorFunc that returns a green-colorized string of the provided input
func Green(s any) string {
	return Colorize(s, GREEN)
}

// GreenBold is a ColorFunc that returns a green-bold-colorized string of the provided input
func GreenBold(s any) string {
	return Colorize(Colorize(s, GREEN), BOLD)
}

// Yellow is a ColorFunc that returns a yellow-colorized string of the provided input
func Yellow(s any) string {
	return Colorize(s, YELLOW)
}

// YellowBold is a ColorFunc that returns a yellow-bold-colorized string of the provided input
func YellowBold(s any) string {
	return Colorize(Colorize(s, YELLOW), BOLD)
}

// Blue is a ColorFunc that returns a blue-colorized string of the provided input
func Blue(s any) string {
	return Colorize(s, BLUE)
}

// BlueBold is a ColorFunc that returns a blue-bold-colorized string of the provided input
func BlueBold(s any) string {
	return Colorize(Colorize(s, BLUE), BOLD)
}

// Magenta is a ColorFunc that returns a magenta-colorized string of the provided input
func Magenta(s any) string {
	return Colorize(s, MAGENTA)
}

// MagentaBold is a ColorFunc that returns a magenta-bold-colorized string of the provided input
func MagentaBold(s any) string {
	return Colorize(Colorize(s, MAGENTA), BOLD)
}

// Cyan is a ColorFunc that returns a cyan-colorized string of the provided input
func Cyan(s any) string {
	return Colorize(s, CYAN)
}

// CyanBold is a ColorFunc that returns a cyan-bold-colorized string of the provided input
func CyanBold(s any) string {
	return Colorize(Colorize(s, CYAN), BOLD)
}

// White is a ColorFunc that returns a white-colorized string of the provided input
func White(s any) string {
	return Colorize(s, WHITE)
}

// WhiteBold is a ColorFunc that returns a white-bold-colorized string of the provided input
func WhiteBold(s any) string {
	return Colorize(Colorize(s, WHITE), BOLD)
}

// Bold is a ColorFunc that returns a bolded string of the provided input
func Bold(s any) string {
	return Colorize(s, BOLD)
}

// DarkGray is a ColorFunc that returns a dark-gray-colorized string of the provided input
func DarkGray(s any) string {
	return Colorize(s, DARK_GRAY)
}

// DarkGrayBold is a ColorFunc that returns a dark-gray-bold-colorized string of the provided input
func DarkGrayBold(s any) string {
	return Colorize(Colorize(s, DARK_GRAY), BOLD)
}
