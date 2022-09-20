package tui

import (
	"fmt"
	"net/url"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rog-golang-buddies/rapidmidiex/internal/suid"
	"github.com/rog-golang-buddies/rapidmidiex/ui/terminal/tui/jamui"
	"github.com/rog-golang-buddies/rapidmidiex/ui/terminal/tui/lobbyui"
)

// ********
// Code heavily based on "Project Journal"
// https://github.com/bashbunni/pjs
// https://www.youtube.com/watch?v=uJ2egAkSkjg&t=319s
// ********

type Session struct {
	Id suid.UUID `json:"id"`
	// UserCount int    `json:"userCount"`
}

type appView int

const (
	jamView appView = iota
	lobbyView
)

type mainModel struct {
	curView      appView
	lobby        tea.Model
	jam          tea.Model
	RESTendpoint string
	WSendpoint   string
}

func NewModel(serverHostURL string) (mainModel, error) {
	wsURL, err := url.Parse(serverHostURL)
	if err != nil {
		return mainModel{}, err
	}
	wsURL.Scheme = "ws"

	return mainModel{
		curView:      lobbyView,
		lobby:        lobbyui.New(),
		jam:          jamui.New(),
		RESTendpoint: serverHostURL + "/api/v1",
		WSendpoint:   wsURL.String() + "/ws",
	}, nil
}

func (m mainModel) Init() tea.Cmd {
	return lobbyui.FetchSessions(m.RESTendpoint)
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	// Handle incoming messages
	switch msg := msg.(type) {

	case tea.KeyMsg:
		// Ctrl+c exits. Even with short running programs it's good to have
		// a quit key, just incase your logic is off. Users will be very
		// annoyed if they can't exit.
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	}

	// Call sub-model Updates
	switch m.curView {
	case lobbyView:
		newLobby, newCmd := m.lobby.Update(msg)
		lobbyModel, ok := newLobby.(lobbyui.Model)
		if !ok {
			panic("could not perform assertion on lobbyui model")
		}
		m.lobby = lobbyModel
		cmd = newCmd
	case jamView:
		newJam, newCmd := m.jam.Update(msg)
		jamModel, ok := newJam.(jamui.Model)
		if !ok {
			panic("could not perform assertion on jamui model")
		}
		m.jam = jamModel
		cmd = newCmd
	}
	// Run all commands from sub-model Updates
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)

}

func (m mainModel) View() string {
	switch m.curView {
	case jamView:
		return m.jam.View()
	default:
		return m.lobby.View()
	}
}

func Run() {
	// TODO: Get from args, user input, or env
	const serverHostURL = "http://localhost:9003"
	m, err := NewModel(serverHostURL)
	if err != nil {
		bail(err)
	}

	if err := tea.NewProgram(m).Start(); err != nil {
		bail(err)
	}
}

func bail(err error) {
	if err != nil {
		fmt.Printf("Uh oh, there was an error: %v\n", err)
		os.Exit(1)
	}
}
