package main

import (
  "fmt"
)

func main() {
  server := NewServer()
  server.Register("getUserByID", func(id int) int { 
    return id 
  })

  err := server.Serve(8080)
  if err != nil {
    fmt.Printf("Server error: %v\n", err)
  }
}
