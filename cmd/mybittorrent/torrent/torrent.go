package torrent

type InfoData struct {
	Length       int    `bencode:"length"`
	Name         string `bencode:"name"`
	Piece_length int    `bencode:"piece length"`
	Pieces       string `bencode:"pieces"`
}
type Torrent struct {
	Announce string   `bencode:"announce"`
	Info     InfoData `bencode:"info"`
}

type TrackerResponse struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}
