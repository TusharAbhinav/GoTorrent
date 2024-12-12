package magnet

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"time"

	infoCommand "github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/info"
	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/peers"
	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/tcp"
	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/torrent"
	"github.com/jackpal/bencode-go"
)

type extensionMsg struct {
	M map[string]interface{} `bencode:"m"`
}
type requestMsgPayload struct {
	Msg_type int `bencode:"msg_type"`
	Piece    int `bencode:"piece"`
}

func ParseMagnetLinks(magnetLink string) (string, string) {
	infoHashPattern := `xt=urn:btih:([a-fA-F0-9]{40}|[a-zA-Z2-7]{32})`
	trackerPattern := `tr=([^&]+)`
	trackerURL := ""

	reTracker := regexp.MustCompile(trackerPattern)
	trackerURLs := reTracker.FindAllStringSubmatch(magnetLink, -1)
	for _, match := range trackerURLs {
		decodedURL, err := url.QueryUnescape(match[1])
		if err != nil {
			fmt.Println(err)
			return "", ""
		}
		trackerURL = decodedURL
		fmt.Println("Tracker URL:", trackerURL)
	}

	reInfoHash := regexp.MustCompile(infoHashPattern)
	infoHash := reInfoHash.FindStringSubmatch(magnetLink)
	if len(infoHash) > 1 {
		fmt.Println("Info Hash:", infoHash[1])
	}
	return trackerURL, infoHash[1]
}

func MagnetHandshake(magnetLink string) {
	trackerURL, infoHash := ParseMagnetLinks(magnetLink)
	byteInfoHash, _ := hex.DecodeString(infoHash)
	var infoHashArray [20]byte
	copy(infoHashArray[:], byteInfoHash)

	peerList, err := peers.FetchPeersFromTracker(trackerURL, infoHashArray, nil)
	if err != nil || len(peerList) == 0 {
		fmt.Println("Error fetching peers or no peers available:", err)
		return
	}
	peerTCPAddr, err := net.ResolveTCPAddr("tcp", peerList[0])
	if err != nil {
		fmt.Println("Error resolving TCP address:", err)
		return
	}

	tcpConn, err := net.DialTCP("tcp", nil, peerTCPAddr)
	if err != nil {
		fmt.Println("Error establishing TCP connection:", err)
		return
	}
	defer tcpConn.Close()
	t := time.Now().Add(7 * time.Second)
	tcpConn.SetDeadline(t)
	peerID := tcp.CompleteHandshake(tcpConn, infoHashArray)
	fmt.Println("Peer ID:", peerID)
	sendExtensionHandshake(tcpConn, infoHash)
}

func sendExtensionHandshake(tcpConn *net.TCPConn, infoHash string) {
	var peerMetaDataExtensionID int
	requestMsgSent := false

	for {
		messageLength := make([]byte, 4)
		_, err := io.ReadFull(tcpConn, messageLength)
		if err != nil {
			fmt.Println("Error reading message length:", err)
			return
		}
		length := binary.BigEndian.Uint32(messageLength)
		if length == 0 {
			fmt.Println("Keep alive message received")
			return
		}

		messageID := make([]byte, 1)
		_, err = io.ReadFull(tcpConn, messageID)
		if err != nil {
			fmt.Println("Error reading message ID:", err)
			return
		}

		id := uint8(messageID[0])
		switch id {
		case 5:
			fmt.Println("Received bitfield message")
			payload := make([]byte, length-1)
			_, err := io.ReadFull(tcpConn, payload)
			if err != nil {
				fmt.Println("Error reading bitfield payload:", err)
				return
			}

			bencodedDict := map[string]interface{}{
				"m": map[string]uint8{"ut_metadata": 20},
			}
			var bencodedDictBytesBuffer bytes.Buffer
			err = bencode.Marshal(&bencodedDictBytesBuffer, bencodedDict)
			if err != nil {
				fmt.Println("Error encoding bencoded dictionary:", err)
				return
			}

			extensionPayload := bencodedDictBytesBuffer.Bytes()
			messageLen := len(extensionPayload) + 2
			extensionHandshake := make([]byte, 4+messageLen)
			binary.BigEndian.PutUint32(extensionHandshake[0:4], uint32(messageLen))
			extensionHandshake[4] = 20
			extensionHandshake[5] = 0
			copy(extensionHandshake[6:], extensionPayload)
			fmt.Println("Sending extension handshake...")
			_, err = tcpConn.Write(extensionHandshake)
			if err != nil {
				fmt.Println("Error sending extension handshake:", err)
			}

		case 20:
			payload := make([]byte, length-1)
			_, err = io.ReadFull(tcpConn, payload)
			if err != nil {
				fmt.Println("Error reading extension message payload:", err)
				return
			}

			extensionMsgID := payload[0]
			dict := payload[1:]
			buf := bytes.NewReader(dict)

			if extensionMsgID == 0 {
				extensionMsg := extensionMsg{}
				err := bencode.Unmarshal(buf, &extensionMsg)
				if err != nil {
					fmt.Println("Error unmarshaling extension message:", err)
					return
				}

				if metadataExtID, ok := extensionMsg.M["ut_metadata"].(int); ok {
					peerMetaDataExtensionID = metadataExtID
					fmt.Println("Peer Metadata Extension ID:", peerMetaDataExtensionID)

					requestMsgPayload := requestMsgPayload{
						Msg_type: 0,
						Piece:    0,
					}
					var requestMsgPayloadBuf bytes.Buffer
					err = bencode.Marshal(&requestMsgPayloadBuf, requestMsgPayload)
					if err != nil {
						fmt.Println("Error marshaling request payload:", err)
						return
					}

					payloadLength := len(requestMsgPayloadBuf.Bytes()) + 2
					requestMsg := make([]byte, 4+payloadLength)
					binary.BigEndian.PutUint32(requestMsg[:4], uint32(payloadLength))
					requestMsg[4] = 20
					requestMsg[5] = uint8(peerMetaDataExtensionID)
					copy(requestMsg[6:], requestMsgPayloadBuf.Bytes())
					fmt.Println("Sending request message...")
					_, err = tcpConn.Write(requestMsg)
					if err != nil {
						fmt.Println("Error sending request message:", err)
						return
					}
					requestMsgSent = true
				} else {
					fmt.Println("Could not extract metadata extension ID")
					return
				}
			} else if requestMsgSent {
				data, err := bencode.Decode(buf)
				if err != nil {
					fmt.Println("Error unmarshaling data message:", err)
					return
				}

				if reflect.TypeOf(data).Kind() == reflect.Map {
					if metadataMap, ok := data.(map[string]interface{}); ok {
						var metadataMapBuf bytes.Buffer
						err = bencode.Marshal(&metadataMapBuf, metadataMap)
						if err != nil {
							fmt.Println("Error encoding metadata map:", err)
							return
						}

						mapBytesLength := metadataMapBuf.Len()
						metadataPieceContentsBuf := bytes.NewReader(dict[mapBytesLength:])

						var metadataPieceContents torrent.InfoData
						err = bencode.Unmarshal(metadataPieceContentsBuf, &metadataPieceContents)
						if err != nil {
							fmt.Println("Error unmarshaling metadata piece contents:", err)
							fmt.Printf("Problematic bytes: %x\n", dict[1+mapBytesLength:])
							return
						}

						hash, err := infoCommand.GenerateInfoHash(metadataPieceContents)
						if err != nil {
							fmt.Println(err)
							return
						}
						if infoHash == hex.EncodeToString(hash[:]) {

							fmt.Println("Length:", metadataPieceContents.Length)
							fmt.Println("Info Hash:", hex.EncodeToString(hash[:]))
							fmt.Println("Piece Length:", metadataPieceContents.Piece_length)
							fmt.Println("Piece Hashes:", hex.EncodeToString([]byte(metadataPieceContents.Pieces)))
						}
					}
				}
			}
		}
	}
}
