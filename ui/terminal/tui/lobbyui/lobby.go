package lobbyui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

// Message types
type statusMsg int

type errMsg struct{ err error }

type Session struct {
	Id string `json:"sessionId"` // TODO: Need to fix the API to return "id"
	// UserCount int    `json:"userCount"`
}
type jamsResp struct {
	Sessions []Session `json:"sessions"`
}

// For messages that contain errors it's often handy to also implement the
// error interface on the message.
func (e errMsg) Error() string { return e.err.Error() }

// Commands
func FetchSessions(baseURL string) tea.Cmd {
	return func() tea.Msg {
		// Create an HTTP client and make a GET request.
		c := &http.Client{Timeout: 10 * time.Second}
		res, err := c.Get(baseURL + "/jam")
		if err != nil {
			// There was an error making our request. Wrap the error we received
			// in a message and return it.
			return errMsg{err}
		}
		// We received a response from the server.
		// Return the HTTP status code
		// as a message.
		if res.StatusCode >= 400 {
			return statusMsg(res.StatusCode)
		}
		decoder := json.NewDecoder(res.Body)
		var resp jamsResp
		decoder.Decode(&resp)
		return resp
	}
}

type Model struct {
	sessions []Session
	jamTable table.Model
	status   int
	err      error
}

// Init needed to satisfy Model interface. It doesn't seem to be called on sub-models.
func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case statusMsg:
		// The server returned a status message. Save it to our model. Also
		// tell the Bubble Tea runtime we want to exit because we have nothing
		// else to do. We'll still be able to render a final view with our
		// status message.
		m.status = int(msg)
		cmds = append(cmds, tea.Quit)
	case jamsResp:
		m.sessions = msg.Sessions
		m.jamTable = makeJamsTable(m)

		cmds = append(cmds, tea.Quit)
	case errMsg:
		// There was an error. Note it in the model. And tell the runtime
		// we're done and want to quit.
		m.err = msg
		cmds = append(cmds, tea.Quit)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	// If there's an error, print it out and don't do anything else.
	if m.err != nil {
		return fmt.Sprintf("\nWe had some trouble: %v\n\n", m.err)
	}

	// Tell the user we're doing something.
	s := fmt.Sprintln("Fetching Jam Sessions...")
	// When the server responds with a status, add it to the current line.
	if m.status > 0 {
		s += fmt.Sprintf("%d %s!", m.status, http.StatusText(m.status))
	}

	if m.sessions != nil {
		s += baseStyle.Render(m.jamTable.View()) + "\n"
	}

	// Send off whatever we came up with above for rendering.
	return "\n" + s + "\n\n"
}

func New() tea.Model {
	return Model{}
}

// https://github.com/rog-golang-buddies/rapidmidiex-research/issues/9#issuecomment-1204853876
func makeJamsTable(m Model) table.Model {
	columns := []table.Column{
		{Title: "Name", Width: 15},
		{Title: "ID", Width: 15},
		{Title: "Players", Width: 10},
		// {Title: "Latency", Width: 4},
	}

	rows := make([]table.Row, 0)

	for _, s := range m.sessions {
		row := table.Row{"Name Here", s.Id, "0"}
		rows = append(rows, row)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return t
}
