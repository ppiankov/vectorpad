package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("vectorpad", version)
		return
	}
	fmt.Println("vectorpad — semantic-preserving editor for reasoning intent")
}
