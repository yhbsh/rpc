package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"reflect"
	"time"
)

type Server struct {
	procs map[string]reflect.Value
}

func NewServer() *Server {
	return &Server{procs: make(map[string]reflect.Value)}
}

func (s *Server) Register(name string, procedure interface{}) {
	s.procs[name] = reflect.ValueOf(procedure)
}

func (s *Server) Serve(port int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	defer listener.Close()
	fmt.Printf("Server listening on port %d\n", port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	for {
		start := time.Now()
		procName, err := readString(conn)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Error reading procedure name: %v\n", err)
			}
			return
		}
		proc, ok := s.procs[procName]
		if !ok {
			fmt.Printf("Unknown procedure: %s\n", procName)
			continue
		}
		args, err := readArgs(conn, proc.Type().NumIn())
		if err != nil {
			fmt.Printf("Error reading arguments: %v\n", err)
			continue
		}
		results := proc.Call(args)
		if len(results) > 0 {
			err = writeFlexibleResult(conn, results[0].Interface())
			if err != nil {
				fmt.Printf("Error writing result: %v\n", err)
				return
			}
		}
		elapsed := time.Since(start)
		fmt.Printf("Request for %s processed in %v\n", procName, elapsed)
	}
}

func readString(r io.Reader) (string, error) {
	var length int64
	err := binary.Read(r, binary.BigEndian, &length)
	if err != nil {
		return "", err
	}
	buffer := make([]byte, length)
	_, err = io.ReadFull(r, buffer)
	if err != nil {
		return "", err
	}
	return string(buffer), nil
}

func readArgs(r io.Reader, count int) ([]reflect.Value, error) {
	args := make([]reflect.Value, count)
	for i := 0; i < count; i++ {
		arg, err := readString(r)
		if err != nil {
			return nil, err
		}
		args[i] = reflect.ValueOf(arg)
	}
	return args, nil
}

func writeFlexibleResult(w io.Writer, result interface{}) error {
	var resultBytes []byte
	var err error

	switch v := result.(type) {
	case string, int, float64, bool:
		// For simple types, convert to string
		resultBytes = []byte(fmt.Sprintf("%v", v))
	default:
		// For complex types, marshal to JSON
		resultBytes, err = json.Marshal(v)
		if err != nil {
			return err
		}
	}

	err = binary.Write(w, binary.BigEndian, int64(len(resultBytes)))
	if err != nil {
		return err
	}
	_, err = w.Write(resultBytes)
	return err
}

func main() {
	server := NewServer()
	
	server.Register("Echo", func(s string) string { return s })
	server.Register("Add", func(a, b string) int {
		var x, y int
		fmt.Sscanf(a, "%d", &x)
		fmt.Sscanf(b, "%d", &y)
		return x + y
	})
	
	server.Register("ComplexData", func() interface{} {
		return map[string]interface{}{
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
