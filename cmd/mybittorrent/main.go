package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/torrent"
	bencode "github.com/jackpal/bencode-go"
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

func decodeBencode(bencodedString string) (interface{}, error) {
	data, err := bencode.Decode(strings.NewReader(bencodedString))
	return data, err
}

func decodeCommand(bencodedValue string) {
	decoded, err := decodeBencode(bencodedValue)
	if err != nil {
		fmt.Println("Error decoding bencoded value:", err)
		return
	}
	jsonOutput, _ := json.Marshal(decoded)
	fmt.Println(string(jsonOutput))
}

func loadTorrentFile(filePath string) (*torrent.Torrent, error) {
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

func generateInfoHash(info torrent.InfoData) ([20]byte, error) {
	var infoBuff bytes.Buffer
	err := bencode.Marshal(&infoBuff, info)
	if err != nil {
		return [20]byte{}, fmt.Errorf("error encoding info dictionary: %v", err)
	}
	hash := sha1.Sum(infoBuff.Bytes())
	return hash, nil
}

func infoCommand(bencodedValue string) {
	metadata, err := loadTorrentFile(bencodedValue)
	if err != nil {
		fmt.Println(err)
		return
	}
	infoHash, err := generateInfoHash(metadata.Info)
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

func fetchPeersFromTracker(trackerURL string, infoHash [20]byte, metadata *torrent.Torrent) ([]string, error) {
	baseURL, _ := url.Parse(trackerURL)
	params := url.Values{}
	params.Add("info_hash", string(infoHash[:]))
	params.Add("peer_id", "tgtwvrxkbjmspmivqnsj")
	params.Add("port", strconv.Itoa(6881))
	params.Add("uploaded", "48")
	params.Add("downloaded", "48")
	params.Add("left", strconv.Itoa(metadata.Info.Length))
	params.Add("compact", "1")
	baseURL.RawQuery = params.Encode()

	resp, err := http.Get(baseURL.String())
	if err != nil {
		return nil, fmt.Errorf("error fetching trackerURL %s: %v", trackerURL, err)
	}
	defer resp.Body.Close()

	trackerStruct := torrent.TrackerResponse{}
	if err := bencode.Unmarshal(resp.Body, &trackerStruct); err != nil {
		return nil, fmt.Errorf("error unmarshalling tracker response: %v", err)
	}

	peerData := []byte(trackerStruct.Peers)
	var peers []string
	for i := 0; i < len(peerData); i += 6 {
		ip := fmt.Sprintf("%d.%d.%d.%d", peerData[i], peerData[i+1], peerData[i+2], peerData[i+3])
		port := binary.BigEndian.Uint16(peerData[i+4 : i+6])
		peers = append(peers, fmt.Sprintf("%s:%d", ip, port))
	}
	return peers, nil
}

func peersCommand(bencodedValue string) {
	metadata, err := loadTorrentFile(bencodedValue)
	if err != nil {
		fmt.Println(err)
		return
	}
	infoHash, err := generateInfoHash(metadata.Info)
	if err != nil {
		fmt.Println(err)
		return
	}

	peers, err := fetchPeersFromTracker(metadata.Announce, infoHash, metadata)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Peers:", peers)
}

func connectTCP(bencodedValue string, peerAddr string) {
	metadata, err := loadTorrentFile(bencodedValue)
	if err != nil {
		fmt.Println(err)
		return
	}
	infoHash, err := generateInfoHash(metadata.Info)
	if err != nil {
		fmt.Println(err)
		return
	}
	peerTCPAddr, err := net.ResolveTCPAddr("tcp", peerAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	tcpRequest := torrent.TCPRequest{Length: 19, Protocol: [19]byte{}, Reserve: [8]byte{0}, InfoHash: infoHash, PeerID: [20]byte{}}
	var tcpBuf []byte
	tcpBuf = append(tcpBuf, byte(tcpRequest.Length))
	copy(tcpRequest.Protocol[:], "BitTorrent protocol")
	tcpBuf = append(tcpBuf, tcpRequest.Protocol[:19]...)
	tcpBuf = append(tcpBuf, tcpRequest.Reserve[:8]...)
	tcpBuf = append(tcpBuf, tcpRequest.InfoHash[:20]...)
	copy(tcpRequest.PeerID[:], "tgtwvrxkbjmspmivqnsj")
	tcpBuf = append(tcpBuf, tcpRequest.PeerID[:20]...)

	tcpConn, err := net.DialTCP("tcp", nil, peerTCPAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer tcpConn.Close()
	tcpConn.SetDeadline(time.Now().Add(5 * time.Second))
	_, err = tcpConn.Write(tcpBuf)
	if err != nil {
		fmt.Println(err)
		return
	}
	reader := bufio.NewReader(tcpConn)
	readBuff, err := io.ReadAll(reader)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Peer ID:", hex.EncodeToString(readBuff[48:]))
}

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
		decodeCommand(bencodedValue)
	case "info":
		infoCommand(bencodedValue)
	case "peers":
		peersCommand(bencodedValue)
	case "handshake":
		connectTCP(bencodedValue, os.Args[3])
	default:
		fmt.Println("Unknown command:", command)
	}
}
