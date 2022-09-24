package jamui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

const (
	// In real life situations we'd adjust the document to fit the width we've
	// detected. In the case of this example we're hardcoding the width, and
	// later using the detected width only to truncate in order to avoid jaggy
	// wrapping.
	width = 96

	columnWidth = 30
)

// DocStyle styling for viewports
var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	docStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)

	keyBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "-",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	key = lipgloss.NewStyle().
		Align(lipgloss.Center).
		Border(keyBorder, true).
		BorderForeground(highlight).
		Padding(0, 1)
)

// Message Types
type Entered struct{}

type pianoKey struct {
	noteNumber int    // MIDI note number ie: 72
	name       string // Name of musical note, ie: "C5"
	keyMap     string // Mapped qwerty keyboard key. Ex: "q"
}

type Model struct {
	piano      []pianoKey          // Piano keys. {"q": pianoKey{72, "C5", "q", ...}}
	activeKeys map[string]struct{} // Currently active piano keys
	Socket     *websocket.Conn     // Websocket connection for current Jam Session
	ID         string              // Jam Session ID
}

func New() Model {
	return Model{
		piano: []pianoKey{
			{72, "C5", "q"},
			{74, "D5", "w"},
			{76, "E5", "e"},
			{77, "F5", "r"},
			{79, "G5", "t"},
			{81, "A5", "y"},
			{83, "B5", "u"},
			{84, "C6", "i"},
		},

		activeKeys: make(map[string]struct{}),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:
		switch msg.String() {
		// These keys should exit the program.
		case "ctrl+c":
			return m, tea.Quit
		default:
			fmt.Printf("Key press: %s\n", msg.String())
		}

	// Entered the Jam Session
	case Entered:
		fmt.Println(m)
	}

	return m, nil
}

func (m Model) View() string {
	physicalWidth, _, _ := term.GetSize(int(os.Stdout.Fd()))
	doc := strings.Builder{}

	if physicalWidth > 0 {
		docStyle = docStyle.MaxWidth(physicalWidth)
	}

	// Keyboard
	keyboard := lipgloss.JoinHorizontal(lipgloss.Top,
		key.Render("C5"+"\n\n"+"(q)"),
		key.Render("D5"+"\n\n"+"(w)"),
		key.Render("E5"+"\n\n"+"(e)"),
		key.Render("F5"+"\n\n"+"(r)"),
		key.Render("G5"+"\n\n"+"(t)"),
		key.Render("A5"+"\n\n"+"(y)"),
		key.Render("B5"+"\n\n"+"(u)"),
		key.Render("C6"+"\n\n"+"(i)"),
	)
	doc.WriteString(keyboard + "\n\n")
	return docStyle.Render(doc.String())
}

// Commands
func Enter() tea.Msg {
	return Entered{}
}
