package main

import (
	"fmt"
)

func main() {
	server := NewServer()
	server.Register("echo", func(message string) map[string]any {
		return map[string]any{"message": message}
	})

	err := server.Serve(8080)
	if err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
