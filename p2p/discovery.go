package p2p

import (
	"context"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/util"
)

type discoveryNotifee struct {
	h host.Host
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID == n.h.ID() {
		return
	}
	if n.h.Network().Connectedness(pi.ID) != network.Connected {
		// New connection
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := n.h.Connect(ctx, pi); err == nil {
			// fmt.Printf("Connected to peer: %s\n", pi.ID.ShortString())
		}
	}
}

func SetupDiscovery(h host.Host, dht *dht.IpfsDHT) error {
	// 1. mDNS (Local)
	serviceTag := "shellchat-mdns"
	n := &discoveryNotifee{h: h}
	s := mdns.NewMdnsService(h, serviceTag, n)
	if err := s.Start(); err != nil {
		return err
	}

	// 2. DHT (Global)
	// Advertise our service on the DHT
	routingDiscovery := routing.NewRoutingDiscovery(dht)
	util.Advertise(context.Background(), routingDiscovery, "shellchat-global")

	// Look for peers
	go func() {
		for {
			peerChan, err := routingDiscovery.FindPeers(context.Background(), "shellchat-global")
			if err != nil {
				time.Sleep(time.Minute)
				continue
			}
			for peer := range peerChan {
				if peer.ID == h.ID() {
					continue
				}
				if h.Network().Connectedness(peer.ID) != network.Connected {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					if err := h.Connect(ctx, peer); err == nil {
						// Found global peer
					}
					cancel()
				}
			}
			time.Sleep(time.Minute) // Scan periodically
		}
	}()

	return nil
}
