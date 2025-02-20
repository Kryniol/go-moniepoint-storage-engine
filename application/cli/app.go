package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Kryniol/go-moniepoint-storage-engine/application/container"
)

type app struct {
	c *container.Container
}

func NewApp(c *container.Container) *app {
	return &app{c: c}
}

func (a *app) Run(ctx context.Context) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Storage Engine CLI. Type 'help' for available commands.")
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := parts[0]

		switch command {
		case "put":
			if len(parts) < 2 {
				fmt.Println("Usage: put <key>")
				continue
			}
			key := parts[1]
			fmt.Println("Enter value to store (press Enter to submit):")
			value, err := readInput()
			if err != nil {
				fmt.Println("Error reading input:", err)
				continue
			}
			if err := a.c.Engine.Put(ctx, key, []byte(value)); err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Data stored successfully.")
			}
		case "read":
			if len(parts) < 2 {
				fmt.Println("Usage: read <key>")
				continue
			}
			key := parts[1]
			val, err := a.c.Engine.Read(ctx, key)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}
			displayData(val)
		case "delete":
			if len(parts) < 2 {
				fmt.Println("Usage: delete <key>")
				continue
			}
			key := parts[1]
			if err := a.c.Engine.Delete(ctx, key); err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Key deleted successfully.")
			}
		case "readrange":
			if len(parts) < 3 {
				fmt.Println("Usage: readrange <startKey> <endKey>")
				continue
			}
			startKey, endKey := parts[1], parts[2]
			dataMap, err := a.c.Engine.ReadKeyRange(ctx, startKey, endKey)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}
			for key, reader := range dataMap {
				fmt.Printf("Key: %s\n", key)
				displayData(reader)
			}
		case "batchput":
			fmt.Println("Enter key-value pairs (format: key:data, one per line, end with EOF):")
			data, err := readBatchInput()
			if err != nil {
				fmt.Println("Error reading input:", err)
				continue
			}
			if err := a.c.Engine.BatchPut(ctx, data); err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Batch put successful.")
			}
		case "exit":
			fmt.Println("Exiting CLI.")
			return
		default:
			fmt.Println("Available commands:")
			fmt.Println("  put <key>       - Store data under a key (input from stdin)")
			fmt.Println("  read <key>      - Retrieve and display data for a key")
			fmt.Println("  delete <key>    - Delete a key from storage")
			fmt.Println("  readrange <s> <e> - Read keys in range [s, e]")
			fmt.Println("  batchput        - Store multiple key-value pairs (input from stdin)")
			fmt.Println("  exit            - Quit the CLI")
		}
	}
}

func readInput() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		line := scanner.Text()
		return line, nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no input read")
}

func readBatchInput() (map[string][]byte, error) {
	scanner := bufio.NewScanner(os.Stdin)
	data := make(map[string][]byte)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			fmt.Println("Invalid format, use: key:data")
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		data[key] = []byte(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return data, nil
}

func displayData(data []byte) {
	fmt.Println("Data:", string(data))
}
