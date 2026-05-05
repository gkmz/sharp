package clipboard

import "github.com/atotto/clipboard"

// Write copies text to the system clipboard.
func Write(text string) error {
	return clipboard.WriteAll(text)
}

// Read returns the current system clipboard text.
func Read() (string, error) {
	return clipboard.ReadAll()
}
