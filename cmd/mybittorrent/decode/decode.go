package decode

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackpal/bencode-go"
)

func decodeBencode(bencodedString string) (interface{}, error) {
	data, err := bencode.Decode(strings.NewReader(bencodedString))
	return data, err
}

func DecodeCommand(bencodedValue string) {
	decoded, err := decodeBencode(bencodedValue)
	if err != nil {
		fmt.Println("Error decoding bencoded value:", err)
		return
	}
	jsonOutput, _ := json.Marshal(decoded)
	fmt.Println(string(jsonOutput))
}
