package main

import (
	"encoding/csv"
	"log"
	"os"
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
