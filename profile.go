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

var (
	flagsPath string = "flags.csv"
)

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

type AppConfigSetting struct {
	ConfigDir    string
	ProfilesPath string
}

type AppConfig struct {
	Profiles        []Profile `json:"profiles"`
	LastUsedProfile Profile   `json:"last_profile"`
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
		Profiles: []Profile{},
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
func loadFlags() ([]FlagRow, int) {
	// flagMap := make(map[int][]string)
	var flagRecords []FlagRow

	file, err := os.Open(flagsPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
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
	return flagRecords, len(flagRecords)
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
func saveFlagsCmd(records []FlagRow) tea.Cmd {
	return func() tea.Msg {
		file, _ := os.Create("flags.csv")
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
