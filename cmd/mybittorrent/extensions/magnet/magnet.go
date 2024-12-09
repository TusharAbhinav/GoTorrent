package magnet

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/peers"
	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/tcp"
	"github.com/jackpal/bencode-go"
	"io"
	"net"
	"net/url"
	"regexp"
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
	peerID := tcp.CompleteHandshake(tcpConn, infoHashArray)
	fmt.Println("Peer ID:", peerID)
	sendExtensionHandshake(tcpConn)
}

func sendExtensionHandshake(tcpConn *net.TCPConn) {

	// In the BitTorrent protocol, peers send their extension handshake after the base handshake:
	// 1. Both peers set reserved bits to indicate extension protocol support
	// 2. Both peers send their own extension handshake independently
	// 3. There's no need to wait and receive the other peer's base handshake before sending your extension handshake
	// 4. The extension handshake is sent immediately after the base handshake, signaling support for extension protocol

	// fmt.Println("Attempting to read recieved handshake...")
	// recievedHandShakeBuff := make([]byte, 68)
	// _, err := io.ReadFull(tcpConn, recievedHandShakeBuff)
	// if err != nil {
	//     fmt.Println("Error receiving handshake:", err)
	//     return
	// }
	// fmt.Println("1")
	// reservedBytes := recievedHandShakeBuff[20:28]
	// reserve := binary.BigEndian.Uint64(reservedBytes[:])
	// mask := uint64(1) << 20
	// if reserve&mask == 0 {
	//     fmt.Println("Peer does not support extension protocol")
	//     return
	// }
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
			continue
		}

		messageID := make([]byte, 1)
		_, err = io.ReadFull(tcpConn, messageID)
		if err != nil {
			fmt.Println("Error reading message ID:", err)
			return
		}

		id := uint8(messageID[0])
		requestMsgSent := 0
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
			dict := payload[1:]
			buf := bytes.NewReader(dict)
			extensionMsg := extensionMsg{}
			err := bencode.Unmarshal(buf, &extensionMsg)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("Peer Metadata Extension ID:", extensionMsg.M["ut_metadata"])
			requestMsgPayload := requestMsgPayload{
				Msg_type: 0,
				Piece:    0,
			}
			peerMetaDataExtensionID := extensionMsg.M["ut_metadata"].(int)
			var requestMsgPayloadBuf bytes.Buffer
			err = bencode.Marshal(&requestMsgPayloadBuf, requestMsgPayload)
			if err != nil {
				return
			}
			payloadLength := len(requestMsgPayloadBuf.Bytes()) + 2
			requestMsg := make([]byte, 4+payloadLength)
			binary.BigEndian.PutUint32(requestMsg[:4], uint32(payloadLength))
			requestMsg[4] = 20
			requestMsg[5] = uint8(peerMetaDataExtensionID)
			copy(requestMsg[6:], requestMsgPayloadBuf.Bytes())
			_, err = tcpConn.Write(requestMsg)
			if err != nil {
				fmt.Println(err)
				return
			}
			requestMsgSent = 1
		}
		if requestMsgSent == 1 {
			break
		}

	}
}
