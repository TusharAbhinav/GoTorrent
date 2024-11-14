package tcp

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"

	infoCommand "github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/info"
	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/torrent"
)

func CompleteHandshake(tcpConn *net.TCPConn, infoHash [20]byte) string {
	tcpRequest := torrent.TCPRequest{Length: 19, Protocol: [19]byte{}, Reserve: [8]byte{0}, InfoHash: infoHash, PeerID: [20]byte{}}
	var tcpBuf []byte
	tcpBuf = append(tcpBuf, byte(tcpRequest.Length))
	copy(tcpRequest.Protocol[:], "BitTorrent protocol")
	tcpBuf = append(tcpBuf, tcpRequest.Protocol[:19]...)
	// Set the 20th bit to 1 in the Reserve field to indicate magnet extension support
	reserve := binary.BigEndian.Uint64(tcpRequest.Reserve[:])
	mask := uint64(1) << 20 
	reserve |= uint64(mask)
	binary.BigEndian.PutUint64(tcpRequest.Reserve[:], reserve)

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
func ConnectTCP(bencodedValue string, peerAddr string) *net.TCPConn {
	metadata, err := infoCommand.LoadTorrentFile(bencodedValue)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	infoHash, err := infoCommand.GenerateInfoHash(metadata.Info)
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
	peerID := CompleteHandshake(tcpConn, infoHash)

	fmt.Println("Peer ID:", peerID)
	return tcpConn
}
