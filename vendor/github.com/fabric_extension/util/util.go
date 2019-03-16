package util

import (
	"os"
)

func OpenFile(file string) *os.File {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	return f
}
