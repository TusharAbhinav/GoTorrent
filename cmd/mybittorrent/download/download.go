package download

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"

	infoCommand "github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/info"
	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/peers"
	"github.com/codecrafters-io/bittorrent-starter-go/cmd/mybittorrent/tcp"
)

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

func DownloadPiece(bencodedValue string, downloadPath string, pieceIndex string) []byte {
	metadata, err := infoCommand.LoadTorrentFile(bencodedValue)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	peers := peers.PeersCommand(bencodedValue)
	tcpConn := tcp.ConnectTCP(bencodedValue, peers[2])
	pieceData := make([]byte, 0)
	pieceInd, _ := strconv.Atoi(pieceIndex)

	pieceLength := metadata.Info.Piece_length
	//total number of pieces
	// eg: len(metadata.Info.Pieces)/20 // Number of pieces
	// 60/20 = 3 pieces

	// len(metadata.Info.Pieces)/20 - 1 // Index of last piece
	// 3-1 = 2 (pieces are 0-indexed))
	if pieceInd == len(metadata.Info.Pieces)/20-1 {
		lastPieceLength := metadata.Info.Length % metadata.Info.Piece_length
		if lastPieceLength > 0 {
			pieceLength = lastPieceLength
		}
	}
	totalBlocks := (pieceLength)/(16*1024) + 1
	fmt.Println("total blocks", totalBlocks)

	pieceReceivedIndex := 0
	defer tcpConn.Close()

	for {
		messageLength := make([]byte, 4)
		_, err := io.ReadFull(tcpConn, messageLength)
		if err != nil {
			fmt.Println("error reading messageLength", err)
			return nil
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
			return nil
		}
		id := uint8(messageID[0])
		switch id {
		case 5:
			fmt.Println("Received bitfield message")
			payload := make([]byte, length-1)
			_, err := io.ReadFull(tcpConn, payload)
			if err != nil {
				fmt.Println("error reading bitfieldPayload", err)
				return nil
			}
			interested := []byte{0, 0, 0, 1, 2}
			_, err = tcpConn.Write(interested)
			if err != nil {
				fmt.Println("Error sending interested message:", err)
				return nil
			}

		case 1:
			fmt.Println("Unchoke message received")
			for i := 0; i < totalBlocks; i++ {
				blockSize := 16 * 1024
				if i == totalBlocks-1 {
					blockSize = pieceLength % (16 * 1024)
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
					return nil
				}
			}
		case 7:
			header := make([]byte, 8)
			_, err := io.ReadFull(tcpConn, header)
			if err != nil {
				fmt.Println("Error reading piece header:", err)
				return nil
			}
			index := binary.BigEndian.Uint32(header[0:4])
			if int(index) != pieceInd {
				fmt.Printf("Wrong piece index received. Expected %d, got %d\n", pieceInd, index)
				return nil
			}

			blockSize := 16 * 1024
			if pieceReceivedIndex == totalBlocks-1 {
				blockSize = pieceLength % (16 * 1024)
			}

			dataBuff := make([]byte, blockSize)
			_, err = io.ReadFull(tcpConn, dataBuff)
			if err != nil {
				fmt.Printf("Error reading piece data (block %d): %v\n", pieceReceivedIndex, err)
				return nil
			}

			pieceData = append(pieceData, dataBuff...)
			pieceReceivedIndex++
			fmt.Printf("Received block %d of %d (size: %d bytes)\n", pieceReceivedIndex, totalBlocks, blockSize)

			if pieceReceivedIndex == totalBlocks {
				receivedPieceHash := sha1.Sum(pieceData)
				expectedHash := metadata.Info.Pieces[pieceInd*20 : (pieceInd+1)*20]

				if bytes.Equal(receivedPieceHash[:], []byte(expectedHash)) {
					fmt.Println("Piece hash verified successfully")
					if downloadPath != "" {
						err := savePieceToFile(pieceData, downloadPath)
						if err != nil {
							fmt.Println("Error saving piece to file:", err)
							return nil
						}
						fmt.Println("Piece saved successfully")
					}
					return pieceData
				} else {
					fmt.Println("Piece hash verification failed")
					return nil
				}
			}
		}
	}
}
func DownloadFile(bencodedValue string, downloadPath string) {
	metadata, err := infoCommand.LoadTorrentFile(bencodedValue)
	if err != nil {
		fmt.Println("error opening file", bencodedValue)
		return
	}
	totalPieces := len(metadata.Info.Pieces) / 20
	file := make([]byte, 0)
	for i := 0; i < totalPieces; i++ {
		pieceData := DownloadPiece(bencodedValue, "", strconv.Itoa(i))
		file = append(file, pieceData...)
	}
	err = savePieceToFile(file, downloadPath)
	if err != nil {
		fmt.Println("error saving to ", downloadPath)
		return
	}
	fmt.Println("File Saved successfully")

}