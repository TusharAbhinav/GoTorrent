package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/torrent"
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
	}
	if command == "info" {
		bencodedValue := os.Args[2]
		file, err := os.Open(bencodedValue)
		if err != nil {
			fmt.Printf("error opening file %s", bencodedValue)
			return
		}
		defer file.Close()
		torrentData, err := io.ReadAll(file)
		if err != nil {
			fmt.Printf("error reading file %s", bencodedValue)
			return
		}
		buf := bytes.NewReader(torrentData)
		metadata := torrent.Torrent{}
		bencode.Unmarshal(buf, &metadata)
		fmt.Println("Tracker URL:", metadata.Announce)
		fmt.Println("Length:", metadata.Info.Length)
		infoData := metadata.Info
		var infoBuff bytes.Buffer
		err = bencode.Marshal(&infoBuff, infoData)
		if err != nil {
			fmt.Println(err)
			return
		}
		hash := sha1.Sum(infoBuff.Bytes())
		fmt.Println("Info Hash:", hex.EncodeToString(hash[:]))

	}
}
