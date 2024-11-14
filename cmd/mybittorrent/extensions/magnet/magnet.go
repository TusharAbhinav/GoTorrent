package magnet

import (
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"regexp"

	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/peers"
	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/tcp"
)

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
	peers, err := peers.FetchPeersFromTracker(trackerURL, [20]byte(byteInfoHash), nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Peers",peers)
	peerTCPAddr, err := net.ResolveTCPAddr("tcp", peers[0])

	if err != nil {
		fmt.Println(err)
		return
	}

	tcpConn, err := net.DialTCP("tcp", nil, peerTCPAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer tcpConn.Close()
	peerID := tcp.CompleteHandshake(tcpConn, [20]byte(byteInfoHash))
	fmt.Println("Peer ID:", peerID)
}
