package main

import (
	"fmt"
)

func main() {
	server := NewServer()

	server.Register("echo", func(s string) string { return s })
	server.Register("add", func(a, b string) int {
		var x, y int
		fmt.Sscanf(a, "%d", &x)
		fmt.Sscanf(b, "%d", &y)
		return x + y
	})

	server.Register("json", func() any {
		return map[string]any{
			"name": "John Doe",
			"age":  30,
			"address": map[string]string{
				"street": "123 Main St",
				"city":   "Anytown",
				"state":  "CA",
			},
			"hobbies": []string{"reading", "swimming", "coding"},
		}
	})

	err := server.Serve(8080)
	if err != nil {
		fmt.Printf("Server error: %v\n", err)
	}

}
