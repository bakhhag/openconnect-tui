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
type vpnLogMsg string
type vpnStatusMsg string

// OFF = 0
// CONNECTED = 1
// CONNECTING = 2
// ERROR = other
type focusArea int

const (
	focusOptionBar focusArea = iota
	focusInput
	focusIPList
	focusFlagList
	focusFlagModal
	focusConnect
	focusProfile
	focusProfileCreate
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

	vpnStatusStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			Width(20).
			Height(3)
	vpnConnectedStyle    = vpnStatusStyle.BorderForeground(lipgloss.Color("#2cdb6f"))
	vpnConnectingStyle   = vpnStatusStyle.BorderForeground(lipgloss.Color("#FFA500"))
	vpnDisconnectedStyle = vpnStatusStyle.BorderForeground(lipgloss.Color("#3C3C3C"))

	onIpStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#9170f3"))
	nilDomainStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#fa5a5a"))
	selDomainStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#a891f0"))

	nilProfileTiStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Faint(true)

	activeHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	activeFlagStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#2cdb6f"))
	inactiveFlagStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7e7e7e"))
	onFlagStyle       = lipgloss.NewStyle().Bold(true)
	gap               = strings.Repeat(" ", 4)
	choices           = map[int]string{
		0: "connect",
		1: "dig",
		2: "flags",
		3: "profiles",
		4: "settings",
	}
	choicesLen             = len(choices)
	initialFlags, flagsLen = loadFlags()

	programInstance *tea.Program
)

type model struct {
	spinner spinner.Model

	activeTab       int
	activeOption    int
	activeChoice    int
	activeIP        int
	activeIPLen     int
	activeFlag      int
	activeProfile   int
	activeProfileTi int
	selectedIP      string
	selectedDomain  string

	focus focusArea

	textInput    textinput.Model
	profileInput []*textinput.Model

	digErr       error
	digIPs       []string
	digIsLoading bool
	flags        []FlagRow

	ac       *AppConfigSetting
	config   AppConfig
	profiles map[int]Profile

	vpnConnecting bool
	vpnStatus     string
	vpnLogs       []string
	stopChan      chan struct{}
	doneChan      chan struct{}

	program *tea.Program
}

func initialModel() *model {
	ac := newAppConfig()
	initialConfig, _ := ac.loadProfiles()

	ti := textinput.New()
	ti.Placeholder = "example.com"
	ti.CharLimit = 100
	ti.Width = 25

	pi := make([]*textinput.Model, 5)
	for i := range pi {
		t := textinput.New()
		t.CharLimit = 50
		switch i {
		case 0:
			t.Placeholder = "Profile Name"
			t.Focus()
		case 1:
			t.Placeholder = "Server IP"
		case 2:
			t.Placeholder = "Port"
		case 3:
			t.Placeholder = "Username"
		case 4:
			t.Placeholder = "Password"
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		}
		// t.Prompt = ""
		pi[i] = &t
	}
	s := spinner.New()
	s.Spinner = spinner.Meter
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	return &model{
		activeTab:       0,
		activeOption:    0,
		activeChoice:    0,
		activeIP:        0,
		activeFlag:      0,
		activeProfile:   0,
		activeProfileTi: 0,

		selectedIP:     "",
		selectedDomain: "",
		textInput:      ti,
		profileInput:   pi,
		spinner:        s,
		flags:          initialFlags,
		vpnConnecting:  false,
		vpnStatus:      "0",
		ac:             ac,
		config:         initialConfig,
		profiles:       initialConfig.Profiles,
	}
}

func (m *model) Init() tea.Cmd {
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
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case vpnStatusMsg:
		m.vpnStatus = string(msg)
		return m, nil

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

		case "tab":
			m.activeTab = (m.activeTab + 1) % 2
			m.activeChoice = 0

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
				switch m.activeOption {
				case 0:
					m.focus = focusConnect
				case 1:
					m.focus = focusInput
					m.textInput.Focus()
					return m, textinput.Blink
				case 2:
					m.focus = focusFlagList
				case 3:
					m.focus = focusProfile
				}

			}
		case focusProfile:
			numProfiles := len(m.profiles)
			switch msg.String() {
			case "tab":
				m.activeProfile = 0
				m.focus = focusOptionBar
			case "enter":
				m.focus = focusProfileCreate
				m.activeTab = 2
				clearTextInputs(m.profileInput)
			// case "down":
			// 	m.activeProfile = (m.activeProfile + 1) % numProfiles
			// case "up":
			// 	m.activeProfile = (m.activeProfile - 1 + numProfiles) % numProfiles
			case "down", "up":
				if numProfiles < 1 {
					break
				}
				if msg.String() == "down" {
					m.activeProfile = (m.activeProfile + 1) % numProfiles
				} else {
					m.activeProfile = (m.activeProfile - 1 + numProfiles) % numProfiles

				}
			}
		case focusProfileCreate:
			switch msg.String() {
			case "down":
				m.profileInput[m.activeProfileTi].Blur()
				m.activeProfileTi = (m.activeProfileTi + 1) % 5
				m.profileInput[m.activeProfileTi].Focus()

			case "up":
				m.profileInput[m.activeProfileTi].Blur()
				m.activeProfileTi = (m.activeProfileTi - 1 + 5) % 5
				m.profileInput[m.activeProfileTi].Focus()
			case "tab", "esc":
				clearTextInputs(m.profileInput)
				m.profileInput[m.activeProfileTi].Blur()
				m.activeProfile = 0
				m.activeProfileTi = 0
				m.profileInput[m.activeProfileTi].Focus()

				m.activeTab = 1

				m.focus = focusProfile
			case "enter":
				if tiProfileIsEmpty(m.profileInput) {
					break
				} else {
					addProfile(m.profileInput, m.profiles)
					clearTextInputs(m.profileInput)
					m.profileInput[m.activeProfileTi].Blur()
					m.activeProfile = 0
					m.activeProfileTi = 0
					m.profileInput[m.activeProfileTi].Focus()
					m.activeTab = 1
					m.focus = focusProfile

					return m, saveProfilesCmd(m.ac, m.config)
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
				m.focus = focusConnect
			}
		case focusConnect:
			switch msg.String() {
			case "tab":
				m.focus = focusOptionBar
				m.activeTab = 0
			case "enter":

				if m.selectedIP != "" {
					if m.vpnConnecting == false {
						m.stopChan = make(chan struct{})
						m.doneChan = make(chan struct{})
						m.vpnConnecting = true
						go openconnect(m.program, m.stopChan, m.doneChan, m.selectedIP, m.flags)
					} else {
						if m.stopChan != nil {
							close(m.stopChan)
							m.vpnConnecting = false

						}
					}
				} else {
					m.activeOption = 1
					m.focus = focusInput
					m.textInput.Focus()
					return m, textinput.Blink
				}
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
	} else if m.focus == focusProfileCreate {
		updatedInput, updatedCmd := m.profileInput[m.activeProfileTi].Update(msg)
		m.profileInput[m.activeProfileTi] = &updatedInput
		cmd = updatedCmd

	}

	return m, cmd
}

func (m *model) View() string {
	var renderedTabs = make([]string, choicesLen)
	var renderedCols = make([]string, 3)
	var content string
	for i := 0; i < 2; i++ {
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
			case 3:
				if len(m.profiles) > 0 {
					content = ""
				} else {
					parts = append(parts, "Press [Enter] to create a profile.")
					content = lipgloss.JoinVertical(lipgloss.Left, parts...)

				}
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
	var modalParts []string

	switch m.focus {

	case focusFlagModal:
		title := fmt.Sprintf("set --%s :\n", m.flags[m.activeFlag].Flag)
		modalParts = append(modalParts, title, m.textInput.View())
		modalContent := lipgloss.JoinVertical(lipgloss.Left, modalParts...)
		renderedCols[2] = setFlagModalStyle.Render(modalContent)
	case focusConnect:
		var statusStyle lipgloss.Style
		var title string
		title = fmt.Sprintf("state : %s", m.vpnStatus)
		switch m.vpnStatus {
		case "0":
			statusStyle = vpnDisconnectedStyle
			title = "state : Disconnected"
		case "1":
			statusStyle = vpnConnectedStyle
			title = "state : Connected"
		case "2":
			statusStyle = vpnConnectingStyle
			title = "state : Connecting"
		default:
			statusStyle = vpnDisconnectedStyle
			title = "state : Disconnected"
		}

		modalParts = append(modalParts, title)
		modalContent := lipgloss.JoinVertical(lipgloss.Left, modalParts...)
		renderedCols[2] = statusStyle.Render(modalContent)
	case focusProfileCreate:
		for i, ti := range m.profileInput {
			if len(ti.Value()) == 0 {
				ti.PlaceholderStyle = nilProfileTiStyle
				ti.PromptStyle = nilProfileTiStyle
				ti.Cursor.Style = nilProfileTiStyle
			} else {
				ti.PlaceholderStyle = lipgloss.NewStyle()
				ti.PromptStyle = lipgloss.NewStyle()
				ti.Cursor.Style = lipgloss.NewStyle()
			}
			if i == m.activeProfileTi {
				ti.Prompt = "> "
				modalParts = append(modalParts, ti.View())
			} else {
				ti.Prompt = ""
				modalParts = append(modalParts, ti.View())

			}

			modalContent := lipgloss.JoinVertical(lipgloss.Left, modalParts...)
			renderedCols[2] = setFlagModalStyle.Render(modalContent)
		}
	}
	ui := lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)

	return "\n" + ui + "\n\nPress Tab to switch columns | q to quit\n"
}

func main() {
	if !amIAdmin() {
		runAsAdmin()
		return
	}
	m := initialModel()
	p := tea.NewProgram(m)

	m.program = p
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func clearTextInputs(tiArr []*textinput.Model) {
	for _, ti := range tiArr {
		if ti != nil {
			ti.Reset()
		}
	}
}

func tiProfileIsEmpty(tiArr []*textinput.Model) bool {
	for _, ti := range tiArr {
		if len(strings.TrimSpace(ti.Value())) == 0 {
			return true
		}
	}
	return false
}

func addProfile(tiArr []*textinput.Model, profiles map[int]Profile) {
	numCurr := len(profiles)
	profiles[numCurr] = Profile{
		Name: tiArr[0].Value(),
		IP:   tiArr[1].Value(),
		Port: tiArr[2].Value(),
		User: tiArr[3].Value(),
		Pass: tiArr[4].Value(),
	}

}

func saveProfilesCmd(ac *AppConfigSetting, config AppConfig) tea.Cmd {
	return func() tea.Msg {
		_ = ac.saveProfiles(config)
		return nil
	}
}
