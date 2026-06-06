package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	activeTabStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 0).
			Width(30)
		// UnsetBorderTop()
	inactiveTabStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#3C3C3C")).
				Padding(0, 0).
				Width(20)

	// Active column style: purple border and highlighted header
	tabNames = map[int]string{
		0: "connect",
		1: "dig",
		2: "settings",
	}
	// Text highlights
	activeHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
)

type model struct {
	activeTab    int
	activeOption int
}

func initialModel() model {
	return model{
		activeTab:    0,
		activeOption: 0,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

// 2. Handle key presses
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		// Tab moves to the next column
		case "tab":
			m.activeTab = (m.activeTab + 1) % 2

		// Shift+Tab moves to the previous column
		case "shift+tab":
			m.activeTab = (m.activeTab - 1 + 2) % 2
		case "down":
			m.activeOption = (m.activeOption + 1) % 3
		case "up":
			m.activeOption = (m.activeOption - 1 + 3) % 3
		}
	}
	return m, nil
}

// 3. Render the columns horizontally
func (m model) View() string {
	var renderedTabs [3]string
	var renderedCols [2]string
	var content string
	for i := 0; i < 2; i++ {
		// 2. Wrap the column string in either the active or inactive border style
		if i == 0 {
			for ind := 0; ind < 3; ind++ {
				if ind == m.activeOption {
					renderedTabs[ind] = activeHeaderStyle.Render(tabNames[ind])
				} else {
					renderedTabs[ind] = tabNames[ind]
				}
			}
			content = lipgloss.JoinVertical(lipgloss.Top, renderedTabs[0], renderedTabs[1], renderedTabs[2])
		} else {
			content = ""
		}
		if i == m.activeTab {
			renderedCols[i] = activeTabStyle.Render(content)
		} else {
			renderedCols[i] = inactiveTabStyle.Render(content)
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
