# GoTorrent
[![progress-banner](https://backend.codecrafters.io/progress/bittorrent/76413f70-06a0-4bf7-b5f6-2bce17e39835)](https://app.codecrafters.io/users/codecrafters-bot?r=2qF)


## Overview

This project is a comprehensive, feature-rich BitTorrent client implemented in Golang.It provides a robust implementation of the BitTorrent protocol, supporting various operations from parsing magnet links to downloading files.

## 🌟 Features

- **Magnet Link Parsing**
  - Extract detailed metadata from magnet links
  - Handle complex magnet URI formats
  - Resolve trackers and file information

- **Torrent File Handling**
  - Decode and parse .torrent files
  - Extract comprehensive torrent metadata
  - Support for various torrent file versions

- **Peer Discovery and Management**
  - Fetch peer information from trackers
  - Perform robust peer handshakes
  - Manage peer connections efficiently

- **File Download Capabilities**
  - Download complete files or specific pieces
  - Piece-wise downloading with SHA-1 hash validation
  - Support for large and small torrents

- **Network Communication**
  - Implement BitTorrent wire protocol
  - Handle peer-to-peer communication
  - Manage download and upload streams

## 🛠 Prerequisites

- Golang 1.20 or higher
- Go Modules enabled

## 📦 Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/TusharAbhinav/GoTorrent.git
   cd cmd/mybittorrent
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the application:
   ```bash
   go build -o mybittorrent
   ```

## 🚀 Usage

### Command Line Interface

The BitTorrent client supports multiple commands for various operations:

#### Torrent File Commands
- **Decode Torrent File**
  ```bash
  ./mybittorrent decode /path/to/torrent/file.torrent
  ```

- **Get Torrent Information**
  ```bash
  ./mybittorrent info /path/to/torrent/file.torrent
  ```

- **Fetch Peer Information**
  ```bash
  ./mybittorrent peers /path/to/torrent/file.torrent
  ```

#### Download Commands
- **Download Specific Piece**
  ```bash
  ./mybittorrent download_piece /path/to/torrent/file.torrent piece_index
  ```

- **Download Complete File**
  ```bash
  ./mybittorrent download /path/to/torrent/file.torrent /path/to/output/file
  ```

#### Magnet Link Commands
- **Parse Magnet Link**
  ```bash
  ./mybittorrent magnet_parse "magnet:?xt=urn:btih:..."
  ```

- **Download via Magnet Link**
  ```bash
  ./mybittorrent magnet_download "magnet:?xt=urn:btih:..." /path/to/output/file
  ```

## 📂 Project Structure

```
GoTorrent/
│
├── cmd/                  
│   └── mybittorrent/
│       └── main.go          # Application entry point
│
├── decode/               # Torrent file decoding
│   └── decode.go         # Decoding logic
│
├── download/             # File download management
│   └── download.go       # Download implementation
│
├── extensions/           # Additional protocol extensions
│   └── magnet/
│       └── magnet.go     # Magnet link handling
│
├── info/                 # Torrent file information
│   └── info.go           # Torrent metadata extraction
│
├── peers/                # Peer discovery and management
│   └── peers.go          # Peer-related functionality
│
├── queue/                # Download queue management
│   └── queue.go          # Piece download queuing
│
├── tcp/                  # TCP communication
│   └── tcp.go            # Low-level network communication
│
├── torrent/              # Torrent file processing
│   └── torrent.go        # Core torrent file handling
│
├── go.mod                # Go module definition
├── go.sum                # Dependency lockfile
└── README.md             # Project documentation
```


## ⚠️ Limitations

- Currently supports single-file torrents
- Limited to public trackers
- Basic piece validation

## 🙌 Acknowledgments

- CodeCrafters Challenge
- BitTorrent Protocol Specification
- Open-source BitTorrent community

