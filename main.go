package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
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
	focusFlagList
	focusFlagModal
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
	setFlagModalStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7D56F4")).
				Padding(0, 1).
				Width(20).
				Height(3)
	onIpStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#9170f3"))
	nilDomainStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#fa5a5a"))
	selDomainStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#a891f0"))

	activeHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	activeFlagStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#2cdb6f"))
	inactiveFlagStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7e7e7e"))
	onFlagStyle       = lipgloss.NewStyle().Bold(true)
	gap               = strings.Repeat(" ", 4)
	choices           = map[int]string{
		0: "connect",
		1: "dig",
		2: "flags",
		3: "settings",
	}
	choicesLen             = len(choices)
	initialFlags, flagsLen = loadFlags()
)

type model struct {
	spinner        spinner.Model
	activeTab      int
	activeOption   int
	activeChoice   int
	activeIP       int
	activeIPLen    int
	activeFlag     int
	selectedIP     string
	selectedDomain string
	focus          focusArea
	textInput      textinput.Model
	digErr         error
	digIPs         []string
	digIsLoading   bool
	flags          []FlagRow
}

func initialModel() model {

	ti := textinput.New()
	ti.Placeholder = "example.com"
	ti.CharLimit = 100
	ti.Width = 25

	s := spinner.New()
	s.Spinner = spinner.Meter
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	return model{
		activeTab:      0,
		activeOption:   0,
		activeChoice:   0,
		activeIP:       0,
		activeFlag:     0,
		selectedIP:     "",
		selectedDomain: "",
		textInput:      ti,
		spinner:        s,
		flags:          initialFlags,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

type digResult struct {
	ips []string
	err error
}

func dig(domain string) tea.Cmd {
	return func() tea.Msg {
		ips, err := net.LookupIP(domain)
		if err != nil {
			return digResult{err: err}
		}
		var ipStrings []string
		for _, ip := range ips {
			ipStrings = append(ipStrings, ip.String())
		}
		time.Sleep(1 * time.Second)
		return digResult{ips: ipStrings}
	}
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.digIsLoading {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c", "q":
			// TODO : save flags.csv on quit
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
				} else if m.activeOption == 2 {
					m.focus = focusFlagList
				}

			}
		case focusFlagList:
			switch msg.String() {
			case "down":
				m.activeFlag = (m.activeFlag + 1) % flagsLen
			case "up":
				m.activeFlag = (m.activeFlag - 1 + flagsLen) % flagsLen
			case "tab":
				m.activeFlag = 0
				// add saving loading screen later

				m.focus = focusOptionBar
				return m, saveFlagsCmd(m.flags)
			case "enter":
				var selectedFlag = m.flags[m.activeFlag]
				if selectedFlag.Selected == "0" {
					if strings.HasSuffix(selectedFlag.Flag, "=") {
						m.focus = focusFlagModal
						m.activeTab = 2
						m.textInput.Focus()
						return m, textinput.Blink
					} else {
						setFlagSelected(m.flags, m.activeFlag, true)
					}
				} else {
					setFlagSelected(m.flags, m.activeFlag, false)
					if selectedFlag.Value != "" {
						setFlagValue(m.flags, m.activeFlag, "")
					}
				}
			}
		case focusInput:
			switch msg.String() {
			case "tab":
				m.focus = focusOptionBar
			case "enter":
				domain := m.textInput.Value()
				if domain != "" {
					m.digErr = nil
					m.digIPs = nil
					m.digIsLoading = true
					return m, tea.Batch(dig(domain), m.spinner.Tick)
				}
				// m.focus = focusIPList
			}
		case focusIPList:
			switch msg.String() {
			case "tab":
				m.activeIP = 0
				m.focus = focusOptionBar
			case "esc":
				m.activeIP = 0
				m.focus = focusInput
				m.textInput.Focus()
				return m, textinput.Blink
			case "down":
				m.activeIP = (m.activeIP + 1) % m.activeIPLen
			case "up":
				m.activeIP = (m.activeIP - 1 + m.activeIPLen) % m.activeIPLen
			case "enter":
				m.selectedIP = m.digIPs[m.activeIP]
				m.selectedDomain = m.textInput.Value()
				m.textInput.Reset()
				clear(m.digIPs)
				m.activeOption = 0
			}
		case focusFlagModal:
			switch msg.String() {
			case "tab":
				m.focus = focusFlagList
				m.activeTab = 1
				m.textInput.Reset()
				m.textInput.Blur()
			case "enter":
				setFlagValue(m.flags, m.activeFlag, m.textInput.Value())
				setFlagSelected(m.flags, m.activeFlag, true)
				m.focus = focusFlagList
				m.activeTab = 1
				m.textInput.Reset()
				m.textInput.Blur()
				return m, saveFlagsCmd(m.flags)

			}

		}
	case digResult:
		m.digIsLoading = false
		if msg.err != nil {
			m.digErr = msg.err
			m.digIPs = nil
		} else {
			m.digErr = nil
			m.digIPs = msg.ips
			m.activeIPLen = len(msg.ips)
			// m.selectedIP = 0
			m.textInput.Blur()
			m.focus = focusIPList
		}
	}
	if m.focus == focusInput || m.focus == focusFlagModal {
		m.textInput, cmd = m.textInput.Update(msg)
	}
	return m, cmd
}

// 3. Render the columns horizontally
func (m model) View() string {
	// var choicesLength = len(choices)
	var renderedTabs = make([]string, choicesLen)
	var renderedCols = make([]string, 3)
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
			var parts []string
			switch m.activeOption {
			case 0:
				if m.selectedDomain != "" {
					parts = append(parts, "Server:", selDomainStyle.Render(m.selectedDomain), "IP:", selDomainStyle.Render(m.selectedIP))
				} else {
					parts = append(parts, "Server:", nilDomainStyle.Render("Select via 'dig' tab."))
				}
				content = lipgloss.JoinVertical(lipgloss.Left, parts...)

			case 1:
				parts = append(parts, "DNS Lookup Tool")

				// Domain input box
				parts = append(parts, "\nDomain Name:\n"+m.textInput.View())
				if m.digIsLoading {
					parts = append(parts, "\n"+m.spinner.View()+" Retrieving IPs")
				} else {
					if m.digErr != nil {
						parts = append(parts, "\nError:"+m.digErr.Error())
					} else if len(m.digIPs) > 0 {
						for i, ip := range m.digIPs {
							if i == m.activeIP {
								parts = append(parts, onIpStyle.Render(ip))
							} else {
								parts = append(parts, ip)

							}
						}
					}
				}
				content = lipgloss.JoinVertical(lipgloss.Left, parts...)
			case 2:
				var upperThresh int
				upperThresh = m.activeFlag + 8
				if upperThresh > flagsLen {
					upperThresh = flagsLen
				}
				for i := m.activeFlag; i < upperThresh; i++ {
					if m.flags[i].Selected == "1" {
						var flagString string
						if strings.HasSuffix(m.flags[i].Flag, "=") {
							flagString = m.flags[i].Flag + m.flags[i].Value
						} else {
							flagString = m.flags[i].Flag
						}
						parts = append(parts, activeFlagStyle.Render(flagString))
					} else if m.flags[i].Selected == "0" && i == m.activeFlag {
						parts = append(parts, onFlagStyle.Render(m.flags[i].Flag))
					} else {
						parts = append(parts, inactiveFlagStyle.Render(m.flags[i].Flag))

					}
				}

				content = lipgloss.JoinVertical(lipgloss.Left, parts...)

			default:
				content = ""
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
	if m.focus == focusFlagModal {
		var modalParts []string
		title := fmt.Sprintf("set --%s :\n", m.flags[m.activeFlag].Flag)
		modalParts = append(modalParts, title, m.textInput.View())
		modalContent := lipgloss.JoinVertical(lipgloss.Left, modalParts...)
		renderedCols[2] = setFlagModalStyle.Render(modalContent)
	}
	// 3. Join the three styled column boxes horizontally
	ui := lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)

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
