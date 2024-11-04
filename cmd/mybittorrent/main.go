package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	bencode "github.com/jackpal/bencode-go"
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345

// func decodeString(bencodedString string) (string, error) {
// 	var firstColonIndex int

// 	for i := 0; i < len(bencodedString); i++ {
// 		if bencodedString[i] == ':' {
// 			firstColonIndex = i
// 			break
// 		}
// 	}

// 	lengthStr := bencodedString[:firstColonIndex]

// 	length, err := strconv.Atoi(lengthStr)
// 	if err != nil {
// 		return "", err
// 	}

//		return bencodedString[firstColonIndex+1 : firstColonIndex+1+length], nil
//	}
//
//	func decodeInteger(bencodedString string) (int, error) {
//		length := len(bencodedString)
//		n, err := strconv.Atoi(bencodedString[1 : length-1])
//		if err != nil {
//			return -1, err
//		}
//		return n, nil
//	}
func decodeBencode(bencodedString string) (interface{}, error) {
	data, err := bencode.Decode(strings.NewReader(bencodedString))
	return data, err
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Fprintln(os.Stderr, "Logs from your program will appear here!")

	command := os.Args[1]

	if command == "decode" {

		bencodedValue := os.Args[2]

		decoded, err := decodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
