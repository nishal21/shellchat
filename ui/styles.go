package ui

import "github.com/charmbracelet/lipgloss"

// Retro Color Palette (Green/Amber Monochrome)
var (
	ColorGreen = lipgloss.Color("2") // Standard Terminal Green
	ColorDark  = lipgloss.Color("0") // Black
	ColorGray  = lipgloss.Color("8") // Gray for borders
	ColorAmber = lipgloss.Color("3") // Amber/Yellow for highlights
	ColorRed   = lipgloss.Color("1") // Red for errors
)

// Retro Styles
var (
	// Borders
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorGreen).
			Padding(0, 1)

	// Sidebar
	SidebarStyle = BorderStyle.Copy().
			BorderForeground(ColorGray).
			MarginRight(1)

	// Active Tab/Peer
	ActiveStyle = lipgloss.NewStyle().
			Foreground(ColorDark).
			Background(ColorGreen).
			Bold(true)

	// Inactive
	InactiveStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	// Messages
	SenderStyle = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Bold(true)

	ReceiverStyle = lipgloss.NewStyle().
			Foreground(ColorAmber).
			Bold(true)

	TimeStyle = lipgloss.NewStyle().
			Foreground(ColorGray)

	// Input
	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorGreen).
			Padding(0, 1)
)
