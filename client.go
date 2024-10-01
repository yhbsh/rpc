package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type Client struct {
	conn net.Conn
}

func NewClient(address string) (*Client, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

func (c *Client) Call(procedure string, args ...string) (string, error) {
	// Send procedure name
	err := c.writeString(procedure)
	if err != nil {
		return "", err
	}

	// Send arguments
	for _, arg := range args {
		err = c.writeString(arg)
		if err != nil {
			return "", err
		}
	}

	// Read result
	return c.readString()
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) writeString(s string) error {
	err := binary.Write(c.conn, binary.BigEndian, int64(len(s)))
	if err != nil {
		return err
	}
	_, err = c.conn.Write([]byte(s))
	return err
}

func (c *Client) readString() (string, error) {
	var length int64
	err := binary.Read(c.conn, binary.BigEndian, &length)
	if err != nil {
		return "", err
	}

	buffer := make([]byte, length)
	_, err = io.ReadFull(c.conn, buffer)
	if err != nil {
		return "", err
	}

	return string(buffer), nil
}

func main() {
	client, err := NewClient("localhost:8080")
	if err != nil {
		fmt.Printf("Error connecting to server: %v\n", err)
		return
	}
	defer client.Close()

	// Test Echo procedure
	result, err := client.Call("Echo", "Hello, RPC!")
	if err != nil {
		fmt.Printf("Error calling Echo: %v\n", err)
	} else {
		fmt.Printf("Echo result: %s\n", result)
	}

	// Test Add procedure
	result, err = client.Call("Add", "5", "3")
	if err != nil {
		fmt.Printf("Error calling Add: %v\n", err)
	} else {
		fmt.Printf("Add result: %s\n", result)
	}
}
