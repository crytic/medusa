package colors

// init will ensure that ANSI coloring is enabled on Windows and Unix systems. Note that ANSI coloring is enabled by
// default on Unix system and Windows needs specific kernel calls for enablement
func init() {
	EnableColor()
}
