package clipboard

import "github.com/atotto/clipboard"

func Write(text string) error {
	return clipboard.WriteAll(text)
}
