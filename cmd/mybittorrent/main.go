package main

import (
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

func peersCommand(bencodedValue string) []string {
	metadata, err := loadTorrentFile(bencodedValue)
	if err != nil {
		fmt.Println(err)
		return []string{}
	}
	infoHash, err := generateInfoHash(metadata.Info)
	if err != nil {
		fmt.Println(err)
		return []string{}
	}

	peers, err := fetchPeersFromTracker(metadata.Announce, infoHash, metadata)
	if err != nil {
		fmt.Println(err)
		return []string{}
	}
	fmt.Println("Peers:", peers)
	return peers
}
func completeHandshake(tcpConn *net.TCPConn, infoHash [20]byte) string {
	tcpRequest := torrent.TCPRequest{Length: 19, Protocol: [19]byte{}, Reserve: [8]byte{0}, InfoHash: infoHash, PeerID: [20]byte{}}
	var tcpBuf []byte
	tcpBuf = append(tcpBuf, byte(tcpRequest.Length))
	copy(tcpRequest.Protocol[:], "BitTorrent protocol")
	tcpBuf = append(tcpBuf, tcpRequest.Protocol[:19]...)
	tcpBuf = append(tcpBuf, tcpRequest.Reserve[:8]...)
	tcpBuf = append(tcpBuf, tcpRequest.InfoHash[:20]...)
	copy(tcpRequest.PeerID[:], "tgtwvrxkbjmspmivqnsj")
	tcpBuf = append(tcpBuf, tcpRequest.PeerID[:20]...)
	_, err := tcpConn.Write(tcpBuf)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	peerBuff := make([]byte, 48)
	_, err = io.ReadFull(tcpConn, peerBuff)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	peerID := make([]byte, 20)
	_, err = io.ReadFull(tcpConn, peerID)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return hex.EncodeToString(peerID[:])
}
func connectTCP(bencodedValue string, peerAddr string) *net.TCPConn {
	metadata, err := loadTorrentFile(bencodedValue)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	infoHash, err := generateInfoHash(metadata.Info)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	peerTCPAddr, err := net.ResolveTCPAddr("tcp", peerAddr)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	tcpConn, err := net.DialTCP("tcp", nil, peerTCPAddr)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	// tcpConn.SetDeadline(time.Now().Add(5 * time.Second))
	peerID := completeHandshake(tcpConn, infoHash)

	fmt.Println("Peer ID:", peerID)
	return tcpConn
}
func savePieceToFile(pieceData []byte, downloadPath string) error {
	file, err := os.OpenFile(downloadPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	_, err = file.Write(pieceData)
	if err != nil {
		return fmt.Errorf("error writing piece data to file: %v", err)
	}

	return nil
}
func downloadPiece(bencodedValue string, downloadPath string, pieceIndex string) {
	metadata, err := loadTorrentFile(bencodedValue)
	if err != nil {
		fmt.Println(err)
		return
	}
	peers := peersCommand(bencodedValue)
	tcpConn := connectTCP(bencodedValue, peers[2])
	pieceData := make([]byte, 0)
	totalBlocks := (metadata.Info.Piece_length) / (16 * 1024)
	fmt.Println("total blocks", totalBlocks)
	pieceInd, _ := strconv.Atoi(pieceIndex)
	pieceReceivedIndex := 0
	defer tcpConn.Close()
	for {
		messageLength := make([]byte, 4)
		_, err := io.ReadFull(tcpConn, messageLength)
		if err != nil {
			fmt.Println("error reading messageLength", err)
			return
		}
		length := binary.BigEndian.Uint32(messageLength)
		if length == 0 {
			fmt.Println("Keep alive message received")
			continue
		}
		messageID := make([]byte, 1)
		_, err = io.ReadFull(tcpConn, messageID)
		if err != nil {
			fmt.Println("error reading messageID", err)
			return
		}
		id := uint8(messageID[0])
		switch id {
		case 5:

			fmt.Println("Received bitfield message")
			payload := make([]byte, length-1)
			_, err := io.ReadFull(tcpConn, payload)
			if err != nil {
				fmt.Println("error reading bitfieldPayload", err)
				return
			}
			fmt.Println("payload", payload)
			interested := []byte{0, 0, 0, 1, 2}
			_, err = tcpConn.Write(interested)
			if err != nil {
				fmt.Println("Error sending interested message:", err)
				return
			}

		case 1:
			fmt.Println("Unchoke message received")
			for i := 0; i < totalBlocks; i++ {
				blockSize := 16 * 1024
				if i == totalBlocks-1 {
					lastBlockSize := metadata.Info.Piece_length % (16 * 1024)
					if lastBlockSize > 0 {
						blockSize = lastBlockSize
					}
				}
				request := make([]byte, 17)
				binary.BigEndian.PutUint32(request[0:4], 13)                  // Message length (13 bytes)
				request[4] = 6                                                // Message ID (request)
				binary.BigEndian.PutUint32(request[5:9], uint32(pieceInd))    // Piece index
				binary.BigEndian.PutUint32(request[9:13], uint32(i*16*1024))  // Begin offset
				binary.BigEndian.PutUint32(request[13:17], uint32(blockSize)) // Block length

				_, err = tcpConn.Write(request)
				if err != nil {
					fmt.Printf("Error sending request for block %d: %v\n", i+1, err)
					return
				}
			}
		case 7:
			header := make([]byte, 8)
			_, err := io.ReadFull(tcpConn, header)
			if err != nil {
				fmt.Println("Error reading piece header:", err)
				return
			}
			index := binary.BigEndian.Uint32(header[0:4])
			if int(index) != pieceInd {
				fmt.Printf("Wrong piece index received. Expected %d, got %d\n", pieceInd, index)
				return
			}

			blockSize := 16 * 1024
			if pieceReceivedIndex == totalBlocks-1 {
				blockSize = metadata.Info.Piece_length % (16 * 1024)
				if blockSize == 0 {
					blockSize = 16 * 1024
				}
			}

			dataBuff := make([]byte, blockSize)
			_, err = io.ReadFull(tcpConn, dataBuff)
			if err != nil {
				fmt.Printf("Error reading piece data (block %d): %v\n", pieceReceivedIndex, err)
				return
			}

			pieceData = append(pieceData, dataBuff...)
			pieceReceivedIndex++
			fmt.Printf("Received block %d of %d (size: %d bytes)\n", pieceReceivedIndex, totalBlocks, blockSize)

			if pieceReceivedIndex == totalBlocks {
				receivedPieceHash := sha1.Sum(pieceData)
				expectedHash := metadata.Info.Pieces[pieceInd*20 : (pieceInd+1)*20]

				if bytes.Equal(receivedPieceHash[:], []byte(expectedHash)) {
					fmt.Println("Piece hash verified successfully")
					err := savePieceToFile(pieceData, downloadPath)
					if err != nil {
						fmt.Println("Error saving piece to file:", err)
						return
					}
					fmt.Println("Piece saved successfully")
					return
				} else {
					fmt.Println("Piece hash verification failed")
					return
				}

			}
		}
	}
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
	case "download_piece":
		downloadPiece(os.Args[4], os.Args[3], os.Args[5])
	default:
		fmt.Println("Unknown command:", command)
	}
}
