package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	funcs map[string]any
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Register(name string, proc any) {
	if s.funcs == nil {
		s.funcs = make(map[string]any)
	}
	s.funcs[name] = proc
}

func (s *Server) Serve(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Println("Error starting server:", err)
		return err
	}
	defer ln.Close()

	fmt.Printf("Server listening on %d\n", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	start := time.Now()

	reader := bufio.NewReader(conn)

	message, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Connection closed")
		return
	}

	message = strings.TrimSpace(message)

	parts := strings.SplitN(message, " ", 3)
	if len(parts) < 2 {
		conn.Write([]byte("Bad Request\n"))
		return
	}

	funcName := parts[1]
	args := ""
	if len(parts) > 2 {
		args = parts[2]
	}

	function, exists := s.funcs[funcName]
	if !exists {
		conn.Write([]byte("Function Not Found\n"))
		return
	}

	funcValue := reflect.ValueOf(function)
	funcType := funcValue.Type()

	argVals, err := parseArgs(args, funcType)
	if err != nil {
		conn.Write([]byte("Invalid Arguments\n"))
		return
	}

	funcResults := funcValue.Call(argVals)

	resp, err := formatResponse(funcResults)
	if err != nil {
		conn.Write([]byte("Internal Server Error\n"))
		return
	}

	conn.Write([]byte(fmt.Sprintf("%s\n", resp)))

	elapsed := time.Since(start)
	logger := log.New(log.Writer(), "", log.Ldate|log.Ltime|log.Lmicroseconds)
	logger.Printf("%5d µs", elapsed.Microseconds())
}

func formatResponse(results []reflect.Value) (string, error) {
	if len(results) == 1 {
		result := results[0].Interface()
		if reflect.TypeOf(result).Kind() == reflect.Struct || reflect.TypeOf(result).Kind() == reflect.Map {
			jsonData, err := json.Marshal(result)
			if err != nil {
				return "", err
			}
			return string(jsonData), nil
		}

		return fmt.Sprintf("%v", result), nil
	}

	var resultStrings []string
	for _, result := range results {
		resultStrings = append(resultStrings, fmt.Sprintf("%v", result.Interface()))
	}
	return strings.Join(resultStrings, " "), nil
}

func parseArgs(args string, funcType reflect.Type) ([]reflect.Value, error) {
	argParts := strings.Split(args, " ")
	numArgs := funcType.NumIn()

	if numArgs != len(argParts) {
		return nil, fmt.Errorf("incorrect number of arguments")
	}

	argVals := make([]reflect.Value, numArgs)
	for i := 0; i < numArgs; i++ {
		argType := funcType.In(i)

		switch argType.Kind() {
		case reflect.Int:
			val, err := strconv.Atoi(argParts[i])
			if err != nil {
				return nil, err
			}
			argVals[i] = reflect.ValueOf(val)

		case reflect.String:
			argVals[i] = reflect.ValueOf(argParts[i])

		default:
			return nil, fmt.Errorf("unsupported argument type")
		}
	}

	return argVals, nil
}
