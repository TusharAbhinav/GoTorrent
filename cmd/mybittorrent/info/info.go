package infoCommand

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	 "github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/torrent"
	"github.com/jackpal/bencode-go"
)
func LoadTorrentFile(filePath string) (*torrent.Torrent, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %v", filePath, err)
	}
	defer file.Close()

	torrentData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %v", filePath, err)
	}

	buf := bytes.NewReader(torrentData)
	metadata := torrent.Torrent{}
	if err := bencode.Unmarshal(buf, &metadata); err != nil {
		return nil, fmt.Errorf("error unmarshalling torrent data: %v", err)
	}
	return &metadata, nil
}
func GenerateInfoHash(info torrent.InfoData) ([20]byte, error) {
	var infoBuff bytes.Buffer
	err := bencode.Marshal(&infoBuff, info)
	if err != nil {
		return [20]byte{}, fmt.Errorf("error encoding info dictionary: %v", err)
	}
	hash := sha1.Sum(infoBuff.Bytes())
	return hash, nil
}

func InfoCommand(bencodedValue string) {
	metadata, err := LoadTorrentFile(bencodedValue)
	if err != nil {
		fmt.Println(err)
		return
	}
	infoHash, err := GenerateInfoHash(metadata.Info)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Tracker URL:", metadata.Announce)
	fmt.Println("Length:", metadata.Info.Length)
	fmt.Println("Info Hash:", hex.EncodeToString(infoHash[:]))
	fmt.Println("Piece Length:", metadata.Info.Piece_length)
	fmt.Println("Piece Hashes:", hex.EncodeToString([]byte(metadata.Info.Pieces)))
}
