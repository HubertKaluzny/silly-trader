package drabber

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args
	if len(args) != 4 {
		fmt.Println("Usage: data-grabber <stock|crypto> <ticker> <from-date e.g 01/01/2020> <to-date e.g 01/01/2022>")
		os.Exit(1)
	}

}
