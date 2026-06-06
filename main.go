package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Choice struct {
	Option int
	Name   string
}

type focusArea int

const (
	focusOptionBar focusArea = iota
	focusInput
	focusIPList
)

var (
	activeOptionsTabStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7D56F4")).
				Padding(0, 1).
				Width(10).
				Height(8)
	inactiveOptionsTabStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#3C3C3C")).
				Padding(0, 1).
				Width(10).
				Height(8)
	activeSettingsTabStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7D56F4")).
				Padding(0, 1).
				Width(30).
				Height(8)
	inactiveSettingsTabStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("#3C3C3C")).
					Padding(0, 1).
					Width(30).
					Height(8)

	// Active column style: purple border and highlighted header
	choices = map[int]string{
		0: "connect",
		1: "dig",
	}
	choicesLen = len(choices)
	// Text highlights
	activeHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
)

type model struct {
	activeTab    int
	activeOption int
	activeChoice int
	focus        focusArea
	textInput    textinput.Model
}

func initialModel() model {

	ti := textinput.New()
	ti.Placeholder = "example.com"
	ti.CharLimit = 100
	ti.Width = 25

	return model{
		activeTab:    0,
		activeOption: 0,
		activeChoice: 0,
		textInput:    ti,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

// 2. Handle key presses
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		// Tab moves to the next column
		case "tab":
			m.activeTab = (m.activeTab + 1) % 2
			m.activeChoice = 0

		// Shift+Tab moves to the previous column
		case "shift+tab":
			m.activeTab = (m.activeTab - 1 + 2) % 2
			m.activeChoice = 0
		}
		switch m.focus {
		case focusOptionBar:
			switch msg.String() {
			case "down":
				m.activeOption = (m.activeOption + 1) % choicesLen

			case "up":
				m.activeOption = (m.activeOption - 1 + choicesLen) % choicesLen
			case "tab":
				if m.activeOption == 1 {
					m.focus = focusInput
					m.textInput.Focus()
					return m, textinput.Blink
				}
			}
		case focusInput:
			switch msg.String() {
			case "tab":
				m.focus = focusOptionBar
			}
		}

	}
	if m.focus == focusInput {
		m.textInput, cmd = m.textInput.Update(msg)
	}
	return m, cmd
}

// 3. Render the columns horizontally
func (m model) View() string {
	// var choicesLength = len(choices)
	var renderedTabs = make([]string, choicesLen)
	var renderedCols [2]string
	var content string
	for i := 0; i < 2; i++ {
		// 2. Wrap the column string in either the active or inactive border style
		if i == 0 {
			for ind := 0; ind < choicesLen; ind++ {

				if ind == m.activeOption {
					renderedTabs[ind] = activeHeaderStyle.Render(choices[ind])
				} else {
					renderedTabs[ind] = choices[ind]
				}
			}
			content = lipgloss.JoinVertical(lipgloss.Left, renderedTabs...)
		} else {
			if m.activeOption == 0 {
				content = ""
			} else if m.activeOption == 1 {
				var parts []string
				parts = append(parts, "DNS Lookup Tool")

				// Domain input box
				parts = append(parts, "\nDomain Name:\n"+m.textInput.View())

				content = lipgloss.JoinVertical(lipgloss.Left, parts...)
			}
		}
		if i == 0 {
			if i == m.activeTab {
				renderedCols[i] = activeOptionsTabStyle.Render(content)
			} else {
				renderedCols[i] = inactiveOptionsTabStyle.Render(content)
			}
		} else if i == 1 {
			if i == m.activeTab {
				renderedCols[i] = activeSettingsTabStyle.Render(content)
			} else {
				renderedCols[i] = inactiveSettingsTabStyle.Render(content)
			}
		}
	}

	// 3. Join the three styled column boxes horizontally
	ui := lipgloss.JoinHorizontal(lipgloss.Top, renderedCols[0], renderedCols[1])

	return "\n" + ui + "\n\nPress Tab to switch columns | q to quit\n"
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

// func main() {

// 	cmd := exec.Command("openconnects", "--version")

// 	output, err := cmd.CombinedOutput()

// 	if err != nil {
// 		fmt.Printf("%v", err)
// 	}
// 	fmt.Printf(string(output))
// }
