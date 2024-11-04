package torrent

type infoData struct {
	Length       int    `json:"length"`
	Name         string `json:"name"`
	Piece_length int    `json:"piece length"`
	Pieces       string `json:"pieces"`
}
type Torrent struct {
	Announce string   `json:"announce"`
	Info     infoData `json:"info"`
}
