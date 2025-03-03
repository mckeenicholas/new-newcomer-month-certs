package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
)

type Person struct {
	Name         string       `json:"name"`
	WcaID        *string      `json:"wcaId"`
	Registration *interface{} `json:"registration"`
}

type WCAData struct {
	Persons []Person `json:"persons"`
}

const currentYear = "2025"

func processName(name string, shouldSkip bool) string {
	if shouldSkip && strings.Contains(name, "(") {
		return strings.TrimSpace(name[:strings.Index(name, "(")])
	}
	return name
}

func isEligibleParticipant(person Person, includeAll2025 bool) bool {
	// Must have an active registration
	if person.Registration == nil {
		return false
	}

	// Either a newcomer (no WCA ID) or a 2025 ID if includeAll2025 is true
	hasNoID := person.WcaID == nil
	is2025Competitor := person.WcaID != nil && strings.HasPrefix(*person.WcaID, currentYear)

	return hasNoID || (is2025Competitor && includeAll2025)
}

func getEligiblePersons(data WCAData, includeAll2025, skipNonASCII bool) []string {
	var eligiblePersons []string

	for _, person := range data.Persons {
		if isEligibleParticipant(person, includeAll2025) {
			processedName := processName(person.Name, skipNonASCII)
			eligiblePersons = append(eligiblePersons, processedName)
		}
	}

	sort.Strings(eligiblePersons)
	return eligiblePersons
}

func fetchWCAData(compID string) (*WCAData, error) {
	wcaURL := fmt.Sprintf("https://www.worldcubeassociation.org/api/v0/competitions/%s/wcif/public", compID)

	resp, err := http.Get(wcaURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching competition data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	var data WCAData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}

	return &data, nil
}

func getNames(compID string, includeAll2025, skipNonASCII bool) ([]string, error) {
	data, err := fetchWCAData(compID)
	if err != nil {
		return nil, err
	}

	return getEligiblePersons(*data, includeAll2025, skipNonASCII), nil
}

func getPersonsFromCSV(csvPath string, includeAll2025, skipNonASCII bool) ([]string, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("error opening CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV header: %w", err)
	}

	nameIdx := -1
	wcaIDIdx := -1
	statusIdx := -1

	for i, column := range header {
		switch column {
		case "Name":
			nameIdx = i
		case "WCA ID":
			wcaIDIdx = i
		case "Status":
			statusIdx = i
		}
	}

	if nameIdx == -1 || wcaIDIdx == -1 || statusIdx == -1 {
		return nil, fmt.Errorf("CSV file missing required columns")
	}

	var eligiblePersons []string

	for {
		row, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("error reading CSV row: %w", err)
		}

		if row[statusIdx] != "a" {
			continue
		}

		wcaID := row[wcaIDIdx]
		isNewcomer := wcaID == "null"
		is2025Competitor := strings.HasPrefix(wcaID, currentYear)

		if isNewcomer || (is2025Competitor && includeAll2025) {
			processedName := processName(row[nameIdx], skipNonASCII)
			eligiblePersons = append(eligiblePersons, processedName)
		}
	}

	sort.Strings(eligiblePersons)
	return eligiblePersons, nil
}

func main() {
	compID := flag.String("comp", "", "WCA competition ID")
	all := flag.Bool("all", false, "Include all 2025 IDs, not just newcomers")
	skip := flag.Bool("skip", false, "Omit non-ASCII characters from names")
	outDir := flag.String("out", ".", "Output directory for generated certificates")
	useCSV := flag.String("csv", "", "Use a CSV file instead of fetching data from the WCA API")
	flag.Parse()

	if *useCSV == "" && *compID == "" {
		fmt.Println("Competition ID is required")
		flag.Usage()
		return
	}

	var names []string
	var err error
	var identifier string

	if *useCSV != "" {
		names, err = getPersonsFromCSV(*useCSV, *all, *skip)
		identifier = (*useCSV)[:strings.Index(*useCSV, ".csv")]
	} else {
		fmt.Println("Fetching data...")
		names, err = getNames(*compID, *all, *skip)
		identifier = *compID
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Generating certificates...")
	generate(names, "template.pdf", fmt.Sprintf("%s/newcomer_certs_%s.pdf", *outDir, identifier))
}
