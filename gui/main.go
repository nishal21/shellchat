package main

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"strings"
	"time"

	"shellchat/p2p"
	"shellchat/storage"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"golang.org/x/crypto/argon2"
)

// Retro Theme Colors
var (
	ColorGreen = color.RGBA{0, 255, 0, 255}
	ColorBlack = color.RGBA{0, 0, 0, 255}
	ColorGray  = color.RGBA{50, 50, 50, 255}
)

type chatApp struct {
	a    fyne.App
	w    fyne.Window
	host *p2p.ChatHost

	// UI Components
	msgList  *widget.List
	peerList *widget.List
	msgInput *widget.Entry
	status   *widget.Label

	// Data
	activePeer string
	peers      []string
	messages   []storage.Message
}

func main() {
	a := app.New()
	w := a.NewWindow("ShellChat Mobile")
	w.Resize(fyne.NewSize(400, 700)) // Mobile-ish aspect ratio

	c := &chatApp{
		a:          a,
		w:          w,
		activePeer: "global-room",
		peers:      []string{"global-room"},
	}

	c.showLogin()
	w.ShowAndRun()
}

func (c *chatApp) showLogin() {
	passEntry := widget.NewPasswordEntry()
	passEntry.PlaceHolder = "Master Password"

	loginBtn := widget.NewButton("Unlock Encrypted Link", func() {
		if passEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("password required"), c.w)
			return
		}

		// Init DB
		salt := []byte("shellchat-static-salt")
		key := argon2.IDKey([]byte(passEntry.Text), salt, 1, 64*1024, 4, 32)
		hexKey := fmt.Sprintf("x'%x'", key)

		if err := storage.InitDB(hexKey); err != nil {
			dialog.ShowError(err, c.w)
			return
		}

		c.initP2P()
		c.showChatUI()
	})

	// Retro Styling
	logo := canvas.NewText("SHELLCHAT", ColorGreen)
	logo.TextStyle.Bold = true
	logo.TextSize = 40
	logo.Alignment = fyne.TextAlignCenter

	content := container.NewVBox(
		layout.NewSpacer(),
		logo,
		layout.NewSpacer(),
		passEntry,
		loginBtn,
		layout.NewSpacer(),
	)

	// Background
	bg := canvas.NewRectangle(ColorBlack)
	c.w.SetContent(container.NewMax(bg, content))
}

func (c *chatApp) initP2P() {
	// Initialize Host (Random Port for Mobile)
	// Note: On Mobile, we might need 0 to let OS choose
	h, err := p2p.MakeHost(0, nil)
	if err != nil {
		log.Println("Failed to create host:", err)
		return
	}
	c.host = h

	// Discovery
	go p2p.SetupDiscovery(h.P2PHost, h.DHT)

	// Message Listener
	go func() {
		for msg := range c.host.MsgChan {
			parts := strings.SplitN(msg, "|", 2)
			if len(parts) == 2 {
				peerID := parts[0]
				content := parts[1]
				if storage.DB != nil {
					storage.SaveMessage(peerID, content, time.Now().Unix(), false)
				}

				// Update UI if active
				if peerID == c.activePeer || c.activePeer == "global-room" {
					c.refreshMessages()
				}
				c.addPeer(peerID)
			}
		}
	}()

	// Peer Discovery Listener (Poll DHT peers)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			// In a real implementation we'd hook into discovery events
			// Here we just refresh peer list if needed
		}
	}()
}

func (c *chatApp) showChatUI() {
	// Peers List
	c.peerList = widget.NewList(
		func() int { return len(c.peers) },
		func() fyne.CanvasObject { return widget.NewLabel("peer info") },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(c.peers[id])
		},
	)
	c.peerList.OnSelected = func(id widget.ListItemID) {
		c.activePeer = c.peers[id]
		c.refreshMessages()
	}

	// Message List using List widget for performance
	c.msgList = widget.NewList(
		func() int { return len(c.messages) },
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabel("header"),
				widget.NewLabel("body"),
			)
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			msg := c.messages[id]
			box := o.(*fyne.Container)
			header := box.Objects[0].(*widget.Label)
			body := box.Objects[1].(*widget.Label)

			sender := "THEM"
			if msg.IsSent {
				sender = "YOU"
				header.Alignment = fyne.TextAlignTrailing
				body.Alignment = fyne.TextAlignTrailing
			} else {
				header.Alignment = fyne.TextAlignLeading
				body.Alignment = fyne.TextAlignLeading
			}

			ts := time.Unix(msg.Timestamp, 0).Format("15:04")
			header.SetText(fmt.Sprintf("%s [%s]", sender, ts))
			body.SetText(msg.Content)
		},
	)

	// Input Area
	c.msgInput = widget.NewEntry()
	c.msgInput.PlaceHolder = "Type a message..."
	c.msgInput.OnSubmitted = func(text string) {
		if text == "" {
			return
		}
		c.sendMessage(text)
		c.msgInput.SetText("")
	}

	sendBtn := widget.NewButtonWithIcon("", theme.MailSendIcon(), func() {
		c.msgInput.OnSubmitted(c.msgInput.Text)
	})

	inputContainer := container.NewBorder(nil, nil, nil, sendBtn, c.msgInput)

	// Status Bar
	c.status = widget.NewLabel("Online")

	// Layout
	// Split Container: Peers (Left) | Chat (Right)
	// For Mobile, we might want Tabs or just one view. Let's use Split for Tablet/Desktop
	// and maybe just Chat for small screens? For now, Universal Split.

	chatPanel := container.NewBorder(nil, inputContainer, nil, nil, c.msgList)

	split := container.NewHSplit(c.peerList, chatPanel)
	split.SetOffset(0.3)

	c.w.SetContent(split)

	c.refreshMessages()
}

func (c *chatApp) sendMessage(content string) {
	// handle commands
	if content == "/myid" {
		id := c.host.P2PHost.ID().String()
		c.w.Clipboard().SetContent(id)
		dialog.ShowInformation("ID Copied", id, c.w)
		return
	}

	if strings.HasPrefix(content, "/connect ") {
		addrStr := strings.TrimPrefix(content, "/connect ")
		ma, err := multiaddr.NewMultiaddr(addrStr)
		if err == nil {
			pi, err := peer.AddrInfoFromP2pAddr(ma)
			if err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				c.host.P2PHost.Connect(ctx, *pi)
				c.addPeer(pi.ID.String())
			}
		}
		return
	}

	// Save locally
	storage.SaveMessage(c.activePeer, content, time.Now().Unix(), true)

	// Send P2P
	if c.host != nil {
		for _, p := range c.host.P2PHost.Network().Peers() {
			go c.host.SendMessage(context.Background(), p.String(), content)
		}
	}

	c.refreshMessages()
}

func (c *chatApp) refreshMessages() {
	msgs, _ := storage.GetMessages(c.activePeer, 50)
	c.messages = msgs
	c.msgList.Refresh()
	if len(c.messages) > 0 {
		c.msgList.ScrollTo(len(c.messages) - 1)
	}
}

func (c *chatApp) addPeer(p string) {
	found := false
	for _, pine := range c.peers {
		if pine == p {
			found = true
			break
		}
	}
	if !found {
		c.peers = append(c.peers, p)
		c.peerList.Refresh()
	}
}
