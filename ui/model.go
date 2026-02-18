package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"shellchat/p2p"
	"shellchat/storage"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"golang.org/x/crypto/argon2"
)

type sessionState int

const (
	stateAuth sessionState = iota
	stateChat
)

type Model struct {
	state      sessionState
	passwordIn textinput.Model
	messageIn  textinput.Model
	viewport   viewport.Model
	messages   []storage.Message
	err        error

	// P2P
	host       *p2p.ChatHost
	activePeer string
	peers      []string

	// Layout
	width  int
	height int
}

func InitialModel(host *p2p.ChatHost) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter master password"
	ti.EchoMode = textinput.EchoPassword
	ti.Focus()
	ti.Width = 20
	// Apply Retro Config
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorGreen)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorGreen)

	mi := textinput.New()
	mi.Placeholder = "Type a message... (/myid | /connect <addr>)"
	mi.Focus()
	mi.CharLimit = 1000
	mi.Width = 80
	mi.PromptStyle = lipgloss.NewStyle().Foreground(ColorGreen)
	mi.TextStyle = lipgloss.NewStyle().Foreground(ColorGreen)

	vp := viewport.New(80, 20)
	vp.SetContent("Welcome to ShellChat.\nUnlock to start.")

	return Model{
		state:      stateAuth,
		passwordIn: ti,
		messageIn:  mi,
		viewport:   vp,
		host:       host,
		activePeer: "global-room",
		peers:      []string{"global-room"},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.listenForP2PMessages(),
	)
}

func (m Model) listenForP2PMessages() tea.Cmd {
	return func() tea.Msg {
		if m.host == nil {
			return nil
		}
		msg := <-m.host.MsgChan
		parts := strings.SplitN(msg, "|", 2)
		if len(parts) == 2 {
			peerID := parts[0]
			content := parts[1]
			if storage.DB != nil {
				storage.SaveMessage(peerID, content, time.Now().Unix(), false)
			}
			return p2pMsg{peerID: peerID, content: content}
		}
		return nil
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Responsive resizing
		m.viewport.Width = m.width - 25  // Sidebar approx 20-25
		m.viewport.Height = m.height - 5 // Header/Footer buffer
		m.messageIn.Width = m.width - 5

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.state == stateAuth {

				// Unlock DB
				password := m.passwordIn.Value()
				salt := []byte("shellchat-static-salt")
				key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
				hexKey := fmt.Sprintf("x'%x'", key)

				userConfigDir, err := os.UserConfigDir()
				if err != nil {
					m.err = err
					m.viewport.SetContent(fmt.Sprintf("Error getting config dir: %v", err))
					return m, nil
				}

				if err := storage.InitDB(userConfigDir, hexKey); err != nil {
					m.err = err
					m.viewport.SetContent(fmt.Sprintf("Error: %v\nTry again.", err))
					m.passwordIn.SetValue("")
					return m, nil
				}

				m.state = stateChat
				m.viewport.SetContent("Locating peers...")
				return m, tea.Batch(m.loadHistoryCmd(), m.findPeersCmd())

			} else {
				// Chat or Command
				content := m.messageIn.Value()
				if content == "" {
					return m, nil
				}

				// Command: /myid
				if content == "/myid" {
					var addrs []string
					for _, addr := range m.host.P2PHost.Addrs() {
						addrs = append(addrs, fmt.Sprintf("%s/p2p/%s", addr, m.host.P2PHost.ID()))
					}
					m.viewport.SetContent(fmt.Sprintf("My Addresses:\n%s", strings.Join(addrs, "\n")))
					m.messageIn.SetValue("")
					return m, nil
				}

				// Command: /copyid
				if content == "/copyid" {
					var addrs []string
					for _, addr := range m.host.P2PHost.Addrs() {
						addrs = append(addrs, fmt.Sprintf("%s/p2p/%s", addr, m.host.P2PHost.ID()))
					}
					fullAddr := strings.Join(addrs, "\n")
					if err := clipboard.WriteAll(fullAddr); err != nil {
						m.viewport.SetContent(fmt.Sprintf("Failed to copy: %v", err))
					} else {
						m.viewport.SetContent("Addresses copied to clipboard!")
					}
					m.messageIn.SetValue("")
					return m, nil
				}

				// Command: /help
				if content == "/help" {
					helpText := `
COMMANDS
--------
/myid           - Show your P2P addresses
/copyid         - Copy your addresses to clipboard
/connect <addr> - Connect to a peer by address
/exit           - Return to global room
/clear          - Clear chat history
/quit           - Exit application
/help           - Show this help message
`
					m.viewport.SetContent(helpText)
					m.messageIn.SetValue("")
					return m, nil
				}

				// Command: /exit
				if content == "/exit" {
					m.activePeer = "global-room"
					m.messages, _ = storage.GetMessages(m.activePeer, 50)
					m.updateView()
					m.messageIn.SetValue("")
					return m, nil
				}

				// Command: /quit
				if content == "/quit" {
					return m, tea.Quit
				}

				// Command: /clear
				if content == "/clear" {
					m.messages = []storage.Message{}
					m.updateView()
					m.messageIn.SetValue("")
					return m, nil
				}

				// Command: /connect <multiaddr> OR <peerID>
				if strings.HasPrefix(content, "/connect ") {
					addrStr := strings.TrimPrefix(content, "/connect ")
					m.viewport.SetContent(fmt.Sprintf("Connecting to %s...", addrStr))

					// 1. Try valid Multiaddr
					ma, err := multiaddr.NewMultiaddr(addrStr)
					if err == nil {
						pi, err := peer.AddrInfoFromP2pAddr(ma)
						if err == nil {
							// Connect in background
							go func() {
								ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
								defer cancel()
								if err := m.host.P2PHost.Connect(ctx, *pi); err != nil {
									// In a real app, send callback msg to UI
								}
							}()
							m.addPeer(pi.ID.String())
							m.activePeer = pi.ID.String()
							m.messages, _ = storage.GetMessages(m.activePeer, 50)
							m.updateView()
							m.messageIn.SetValue("")
							return m, nil
						}
					}

					// 2. Try Peer ID (DHT Lookup)
					pid, err := peer.Decode(addrStr)
					if err == nil {
						m.viewport.SetContent(fmt.Sprintf("Looking up Peer ID %s in DHT...", pid.ShortString()))
						go func() {
							ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
							defer cancel()
							pi, err := m.host.DHT.FindPeer(ctx, pid)
							if err != nil {
								// Send error to UI (not implemented in this simplified model, but at least we try)
								return
							}
							if err := m.host.P2PHost.Connect(ctx, pi); err == nil {
								// Connection success, UI will update on next interaction or we could send a Cmd
							}
						}()
						m.addPeer(pid.String())
						m.activePeer = pid.String()
						m.messages, _ = storage.GetMessages(m.activePeer, 50)
						m.updateView()
						m.messageIn.SetValue("")
						return m, nil
					}

					m.viewport.SetContent(fmt.Sprintf("Invalid address or Peer ID: %s", addrStr))
					m.messageIn.SetValue("")
					return m, nil
				}

				// Send
				err := storage.SaveMessage(m.activePeer, content, time.Now().Unix(), true)
				if err != nil {
					m.viewport.SetContent(fmt.Sprintf("Error: %v", err))
					return m, nil
				}

				// P2P Send
				if m.host != nil {
					// Broadcast loop (simplification)
					for _, p := range m.host.P2PHost.Network().Peers() {
						go func(pid string) {
							m.host.SendMessage(context.Background(), pid, content)
						}(p.String())
					}
				}

				m.messageIn.SetValue("")
				return m, m.loadHistoryCmd()
			}
		}

	case p2pMsg:
		m.addPeer(msg.peerID)
		if msg.peerID == m.activePeer || m.activePeer == "global-room" {
			return m, tea.Batch(m.loadHistoryCmd(), m.listenForP2PMessages())
		}
		return m, m.listenForP2PMessages()

	case historyMsg:
		m.messages = msg.messages
		m.updateView()

	case peersFoundMsg:
		// Update peer list from DHT discovery
		for _, p := range msg.peers {
			m.addPeer(p)
		}
	}

	if m.state == stateAuth {
		m.passwordIn, cmd = m.passwordIn.Update(msg)
		return m, cmd
	}

	m.messageIn, cmd = m.messageIn.Update(msg)
	return m, cmd
}

func refreshView(m *Model) {
	var sb strings.Builder
	for _, msg := range m.messages {
		timeStr := TimeStyle.Render(time.Unix(msg.Timestamp, 0).Format("15:04"))
		prefix := ReceiverStyle.Render("THEM")
		if msg.IsSent {
			prefix = SenderStyle.Render("YOU")
		}
		sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", timeStr, prefix, msg.Content))
	}
	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

func (m *Model) addPeer(p string) {
	for _, existing := range m.peers {
		if existing == p {
			return
		}
	}
	m.peers = append(m.peers, p)
}

func (m Model) View() string {
	if m.state == stateAuth {
		errStr := ""
		if m.err != nil {
			errStr = lipgloss.NewStyle().Foreground(ColorRed).Render(m.err.Error())
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			BorderStyle.Render(
				fmt.Sprintf("%s\n\nENCRYPTED LINK\n\n%s\n\n%s", Logo, m.passwordIn.View(), errStr),
			),
		)
	}

	// Split View: Sidebar | Chat
	sidebarContent := lipgloss.NewStyle().Foreground(ColorGreen).Render("CONTACTS\n--------\n")
	for _, p := range m.peers {
		if p == m.activePeer {
			sidebarContent += ActiveStyle.Render("> "+p[:10]+"...") + "\n"
		} else {
			sidebarContent += InactiveStyle.Render("  "+p[:10]+"...") + "\n"
		}
	}

	sidebar := SidebarStyle.Width(20).Height(m.height - 7).Render(sidebarContent)

	chatPane := BorderStyle.Width(m.width - 25).Height(m.height - 7).Render(m.viewport.View())

	inputPane := InputStyle.Width(m.width - 5).Render(m.messageIn.View())

	// Status Bar
	statusMode := "SECURE P2P"
	statusInfo := fmt.Sprintf("ID: %s... | PEERS: %d", m.host.P2PHost.ID().String()[:10], len(m.peers)-1)

	statusBar := lipgloss.NewStyle().
		Width(m.width).
		Background(ColorGreen).
		Foreground(ColorDark).
		Padding(0, 1).
		Render(fmt.Sprintf("%s | %s", statusMode, statusInfo))

	// Join
	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, chatPane)
	body := lipgloss.JoinVertical(lipgloss.Left, main, inputPane)
	return lipgloss.JoinVertical(lipgloss.Left, body, statusBar)
}

func (m *Model) updateView() {
	var sb strings.Builder
	for _, msg := range m.messages {
		timeStr := TimeStyle.Render(time.Unix(msg.Timestamp, 0).Format("15:04"))
		prefix := ReceiverStyle.Render("THEM")
		if msg.IsSent {
			prefix = SenderStyle.Render("YOU")
		}
		sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", timeStr, prefix, msg.Content))
	}
	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

// Commands
type historyMsg struct {
	messages []storage.Message
}
type p2pMsg struct {
	peerID  string
	content string
}
type peersFoundMsg struct {
	peers []string
}
type errMsg struct{ err error }

func (m Model) loadHistoryCmd() tea.Cmd {
	return func() tea.Msg {
		msgs, err := storage.GetMessages(m.activePeer, 50)
		if err != nil {
			return errMsg{err}
		}
		return historyMsg{msgs}
	}
}

func (m Model) findPeersCmd() tea.Cmd {
	return func() tea.Msg {
		// Mock discovery trigger, real logic runs in background in discovery.go
		return nil
	}
}
