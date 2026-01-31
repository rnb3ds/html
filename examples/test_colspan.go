package main

import (
	"fmt"
	"os"

	"github.com/cybergodev/html"
)

func main() {
	content, err := os.ReadFile("/tmp/test_table.html")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	result, err := html.Extract(string(content), html.ExtractConfig{
		TableFormat: "html",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(result.Text)
}
