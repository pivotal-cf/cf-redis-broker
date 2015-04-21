package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	fmt.Println("FAKE AWS CLI")

	outputPath := os.Getenv("FAKE_CLI_OUTPUT_PATH")
	if outputPath != "" {
		f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Fatalf("Error opening file: %s", err.Error())
		}
		_, err = f.WriteString(strings.Join(os.Args[1:], " "))
		if err != nil {
			log.Fatalf("Error writing to file: %s", err.Error())
		}
		f.Close()
	}
}
