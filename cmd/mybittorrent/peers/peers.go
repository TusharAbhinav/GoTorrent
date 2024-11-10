package peers

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	infoCommand "github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/info"
	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/torrent"
	"github.com/jackpal/bencode-go"
)

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

func PeersCommand(bencodedValue string) []string {
	metadata, err := infoCommand.LoadTorrentFile(bencodedValue)
	if err != nil {
		fmt.Println(err)
		return []string{}
	}
	infoHash, err := infoCommand.GenerateInfoHash(metadata.Info)
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
