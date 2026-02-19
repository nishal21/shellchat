package p2p

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

const protocolID = "/shellchat/1.0.0"

// ChatHost handles P2P connections
type ChatHost struct {
	P2PHost host.Host
	DHT     *dht.IpfsDHT
	MsgChan chan string // Channel to send incoming messages to UI
	mu      sync.Mutex
	streams map[string]network.Stream
}

func MakeHost(port int, randomness io.Reader) (*ChatHost, error) {
	// Create identity
	var priv crypto.PrivKey
	var err error

	if randomness == nil {
		priv, _, err = crypto.GenerateKeyPair(crypto.RSA, 2048)
	} else {
		priv, _, err = crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, randomness)
	}
	if err != nil {
		return nil, err
	}

	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))

	// Create libp2p Host with DHT, NAT, and Relay support
	basicHost, err := libp2p.New(
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(priv),
		libp2p.NATPortMap(), // Try to punch through NAT (UPnP)
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(), // Enable Hole Punching instead of AutoRelay (Panic fix)
	)
	if err != nil {
		return nil, err
	}

	// Initialize DHT
	// We use IDht to verify context, but New returns *IpfsDHT
	kademliaDHT, err := dht.New(context.Background(), basicHost)
	if err != nil {
		return nil, err
	}

	// Bootstrap the DHT (connect to public bootstrap nodes)
	if err = kademliaDHT.Bootstrap(context.Background()); err != nil {
		return nil, err
	}

	// Connect to public bootstrap nodes to join the global network
	// Connect to public bootstrap nodes (Background)
	go func() {
		var wg sync.WaitGroup
		for _, peerAddr := range dht.DefaultBootstrapPeers {
			peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // 10s timeout
				defer cancel()
				if err := basicHost.Connect(ctx, *peerinfo); err != nil {
					// fmt.Println(err)
				}
			}()
		}
		wg.Wait()
		// fmt.Println("Bootstrap complete")
	}()

	ch := &ChatHost{
		P2PHost: basicHost,
		DHT:     kademliaDHT,
		MsgChan: make(chan string),
		streams: make(map[string]network.Stream),
	}

	basicHost.SetStreamHandler(protocolID, ch.handleStream)

	return ch, nil
}

func (ch *ChatHost) handleStream(s network.Stream) {
	// Add stream to map
	peerID := s.Conn().RemotePeer().String()
	ch.mu.Lock()
	ch.streams[peerID] = s
	ch.mu.Unlock()

	// Create a buffer stream for non blocking read and write.
	buf := make([]byte, 1024)
	for {
		n, err := s.Read(buf)
		if n > 0 {
			msg := string(buf[:n])
			// Format: "PEER_ID|MESSAGE"
			ch.MsgChan <- fmt.Sprintf("%s|%s", peerID, msg)
		}
		if err != nil {
			if err != io.EOF {
				s.Reset()
			} else {
				s.Close()
			}
			break
		}
	}
}

// SendMessage sends a message to a connected peer
func (ch *ChatHost) SendMessage(ctx context.Context, peerIDStr string, msg string) error {
	ch.mu.Lock()
	s, ok := ch.streams[peerIDStr]
	ch.mu.Unlock()

	if !ok {
		// Try to find and connect
		// This requires resolving peerID string to PeerID
		// And finding address via DHT if not connected
		// For now, assume connected via UI connect command or mDNS.
		return fmt.Errorf("peer not connected")
	}

	_, err := s.Write([]byte(msg))
	return err
}
