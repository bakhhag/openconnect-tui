package main

import (
	"encoding/csv"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
)

const defaultFlags = `flag,selected,value
disable-ipv6,0,
no-dtls,0,
no-xmlpost,0,
base-mtu=,0,
mtu=,0,
sni=,0,google.com
proxy=,0,
no-http-keepalive,0,
`

type FlagRow struct {
	Flag     string
	Selected string
	Value    string
}

type Profile struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	Port string `json:"port"`
	User string `json:"user"`
	Pass string `json:"pass"`
}

type Credential struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}
type AppConfigSetting struct {
	ConfigDir    string
	ProfilesPath string
	FlagsPath    string
}

type AppConfig struct {
	Profiles        []Profile    `json:"profiles"`
	Credentials     []Credential `json:"credentials"`
	LastUsedProfile Profile      `json:"last_profile"`
}

func newAppConfig() *AppConfigSetting {
	baseConfigDir, _ := os.UserConfigDir()
	appConfigDir := filepath.Join(baseConfigDir, "OpenConnect-TUI")
	if _, err := os.Stat(appConfigDir); err != nil {
		os.MkdirAll(appConfigDir, 0700)
	}
	return &AppConfigSetting{
		ConfigDir: appConfigDir,
	}
}

func (ac *AppConfigSetting) loadProfiles() (AppConfig, error) {
	ac.ProfilesPath = filepath.Join(ac.ConfigDir, "config.json")

	config := AppConfig{
		Profiles:        []Profile{},
		LastUsedProfile: Profile{},
	}

	if _, err := os.Stat(ac.ProfilesPath); os.IsNotExist(err) {
		return config, err
	}

	data, err := os.ReadFile(ac.ProfilesPath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	return config, err
}
func (ac *AppConfigSetting) saveProfiles(config AppConfig) error {
	data, _ := json.MarshalIndent(config, "", " ")
	err := os.WriteFile(ac.ProfilesPath, data, 0600)
	return err
}
func (ac *AppConfigSetting) loadFlags() []FlagRow {
	ac.FlagsPath = filepath.Join(ac.ConfigDir, "flags.csv")
	var flagRecords []FlagRow

	file, err := os.OpenFile(ac.FlagsPath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("Failed to open or create flags file: %v", err)
		}
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	if stat.Size() == 0 {
		if _, err := file.WriteString(defaultFlags); err != nil {
			log.Fatalf("Error writing default flags: %v", err)
		}
		if _, err := file.Seek(0, 0); err != nil {
			log.Fatalf("Error rewinding file: %v", err)
		}
	}
	reader := csv.NewReader(file)
	_, err = reader.Read()
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	for _, record := range records {
		// fmt.Println(record)
		if record[1] == "0" {
			flagRecords = append(flagRecords, FlagRow{Flag: record[0], Selected: "0", Value: record[2]})
		} else {
			flagRecords = append(flagRecords, FlagRow{Flag: record[0], Selected: "1", Value: record[2]})

		}

	}
	sort.Slice(flagRecords, func(i, j int) bool {
		return flagRecords[i].Selected > flagRecords[j].Selected
	})
	return flagRecords
}

func setFlagSelected(records []FlagRow, index int, set bool) {
	if set {
		records[index].Selected = "1"
	} else {
		records[index].Selected = "0"
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Selected > records[j].Selected
	})
}

func setFlagValue(records []FlagRow, index int, value string) {
	records[index].Value = value
}
func (ac *AppConfigSetting) saveFlagsCmd(records []FlagRow) tea.Cmd {
	return func() tea.Msg {
		file, _ := os.Create(ac.FlagsPath)
		defer file.Close()
		writer := csv.NewWriter(file)
		defer writer.Flush()
		header := []string{"flag", "selected", "value"}
		writer.Write(header)

		for _, record := range records {
			row := []string{record.Flag, record.Selected, record.Value}
			writer.Write(row)
		}
		return saveFinishedMsg{}
	}
}

type saveFinishedMsg struct{}
