package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	funcs map[string]any
}

func NewServer() *Server {
	return &Server{funcs: make(map[string]any)}
}

func (s *Server) Register(name string, proc any) {
	s.funcs[name] = proc
}

func (s *Server) Serve(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Println("Error starting server:", err)
		return err
	}
	defer ln.Close()

	fmt.Println("[INFO] Registered procedures")
	fmt.Println("-------------------------------------------------------------------------")

	// Extract procedure names into a slice and sort them.
	names := make([]string, 0, len(s.funcs))
	for name := range s.funcs {
		names = append(names, name)
	}
	sort.Strings(names)

	// Iterate over the sorted names to print procedures.
	for _, name := range names {
		proc := s.funcs[name]
		funcType := reflect.TypeOf(proc)
		args := make([]string, funcType.NumIn())
		for i := 0; i < funcType.NumIn(); i++ {
			args[i] = funcType.In(i).String()
		}
		fmt.Printf("[PROC] %-30s | [Args] %v\n", name, args)
	}

	fmt.Println("-------------------------------------------------------------------------")
	fmt.Printf("[INFO] Listening on port %d\n", port)

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
	fmt.Printf("Client %s connected\n", conn.RemoteAddr())
	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Println(err.Error())
			}
			fmt.Printf("Client %s diconnected\n", conn.RemoteAddr())
			return
		}

		message = strings.TrimSpace(message)
		parts := strings.SplitN(message, " ", 3)
		if len(parts) < 2 {
			sendError(conn, "Bad Request")
			continue
		}

		funcName := parts[1]
		args := ""
		if len(parts) > 2 {
			args = parts[2]
		}

		function, exists := s.funcs[funcName]
		if !exists {
			sendError(conn, "Function Not Found")
			continue
		}

		funcValue := reflect.ValueOf(function)
		funcType := funcValue.Type()

		argVals, err := parseArgs(args, funcType)
		if err != nil {
			sendError(conn, err.Error())
			continue
		}

		start := time.Now()
		funcResults := funcValue.Call(argVals)

		resp, err := formatResponse(funcResults)
		if err != nil {
			sendError(conn, err.Error())
			return
		}

		conn.Write([]byte(fmt.Sprintf("%s\n", resp)))

		elapsed := time.Since(start)
		log.Printf("%-15v | %-28s | %s\n", elapsed, funcName, args)
	}
}

func formatResponse(results []reflect.Value) (string, error) {
	if len(results) == 1 {
		result := results[0].Interface()
		jsonData, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(jsonData), nil
	}

	var resultStrings []string
	for _, result := range results {
		resultStrings = append(resultStrings, fmt.Sprintf("%v", result.Interface()))
	}
	return strings.Join(resultStrings, " "), nil
}

func parseArgs(args string, funcType reflect.Type) ([]reflect.Value, error) {
	if args == "" {
		return []reflect.Value{}, nil
	}

	argParts := strings.Split(args, "|")
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
			if argParts[i] == "null" {
				argVals[i] = reflect.Zero(argType)
			} else {
				argVals[i] = reflect.ValueOf(argParts[i])
			}

		case reflect.Ptr:
			elemType := argType.Elem()
			if elemType.Kind() == reflect.String {
				if argParts[i] == "null" {
					argVals[i] = reflect.Zero(argType)
				} else {
					val := argParts[i]
					argVals[i] = reflect.ValueOf(&val)
				}
			} else {
				return nil, fmt.Errorf("unsupported pointer type: %v", elemType.Kind())
			}

		default:
			return nil, fmt.Errorf("unsupported argument type: %v", argType.Kind())
		}
	}

	return argVals, nil
}

func sendError(conn net.Conn, errMsg string) {
	err := map[string]any{"error": errMsg}
	msg, _ := json.Marshal(err)
	msgWithNewline := append(msg, '\n')
	conn.Write(msgWithNewline)
}
