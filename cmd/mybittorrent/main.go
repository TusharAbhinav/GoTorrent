package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/decode"
	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/download"
	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/extensions/magnet"
	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/info"
	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/peers"
	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/tcp"
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

func main() {
	fmt.Fprintln(os.Stderr, "Logs from your program will appear here!")

	if len(os.Args) < 3 {
		fmt.Println("Usage: <command> <bencoded_value>")
		return
	}

	command := os.Args[1]
	bencodedValue := os.Args[2]
	switch command {
	case "decode":
		decode.DecodeCommand(bencodedValue)
	case "info":
		infoCommand.InfoCommand(bencodedValue)
	case "peers":
		peers.PeersCommand(bencodedValue)
	case "handshake":
		tcp.ConnectTCP(bencodedValue, os.Args[3])
	case "download_piece":
		download.DownloadPiece(os.Args[4], os.Args[3], os.Args[5])
	case "download":
		download.DownloadFile(os.Args[4], os.Args[3])
	case "magnet_parse":
		magnet.ParseMagnetLinks(os.Args[2])
	default:
		fmt.Println("Unknown command:", command)
	}
}
