package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p/core/host"
	network "github.com/libp2p/go-libp2p/core/network"
	peer "github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

const dataDir = "/data" // Shared Docker volume

// Save our Peer ID for others
func savePeerID(h host.Host) {
    nodeName := os.Getenv("NODE_NAME")
    if nodeName == "" {
        log.Fatalf("NODE_NAME environment variable not set")
    }

    filename := fmt.Sprintf("%s/%s-id.txt", dataDir, nodeName)

    f, err := os.Create(filename)
    if err != nil {
        log.Println("Failed to save peer ID:", err)
        return
    }
    defer f.Close()

    _, _ = f.WriteString(h.ID().String())
    log.Println("Saved peer ID to file:", filename)
}

// Read peer IDs from /data
func readPeerIDs() map[string]string {
	peers := make(map[string]string)
	files, err := os.ReadDir(dataDir)
	if err != nil {
		log.Println("Failed to read peer IDs:", err)
		return peers
	}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), "-id.txt") {
			name := strings.TrimSuffix(f.Name(), "-id.txt")
			content, _ := os.ReadFile(dataDir + "/" + f.Name())
			peers[name] = strings.TrimSpace(string(content))
		}
	}
	return peers
}

// Handle incoming streams
func handleStream(s network.Stream) {
	log.Println("Got a new stream from:", s.Conn().RemotePeer())
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	msg, _ := rw.ReadString('\n')
	log.Printf("Received: %s", msg)
}

func main() {
	ctx := context.Background()

	// Listen on all interfaces (so other containers can reach us)
	listenAddr, _ := ma.NewMultiaddr("/ip4/0.0.0.0/tcp/4001")
	h, err := libp2p.New(libp2p.ListenAddrs(listenAddr))
	if err != nil {
		log.Fatal(err)
	}

	// Register handler for incoming messages
	h.SetStreamHandler("/chat/1.0.0", handleStream)

	log.Println("Node started. ID:", h.ID().String())
	savePeerID(h)

	// Try connecting to discovered peers
	peers := readPeerIDs()
	for name, pid := range peers {
		if name == os.Getenv("NODE_NAME") { // skip self
			continue
		}
		peerID, err := peer.Decode(pid)
		if err != nil {
			log.Println("Invalid peer ID for", name, ":", err)
			continue
		}
		addr, _ := ma.NewMultiaddr(fmt.Sprintf("/dns4/%s/tcp/4001/p2p/%s", name, peerID.String()))
		info, _ := peer.AddrInfoFromP2pAddr(addr)
		if err := h.Connect(ctx, *info); err != nil {
			log.Println("Failed to connect to", name, ":", err)
		} else {
			log.Println("Connected to", name)
			// Send a hello message
			s, err := h.NewStream(ctx, peerID, "/chat/1.0.0")
			if err == nil {
				rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
				rw.WriteString(fmt.Sprintf("Hello from %s!\n", os.Getenv("NODE_NAME")))
				rw.Flush()
			}
		}
	}

	select {} // keep running
}
