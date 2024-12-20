package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Cage struct {
	IP     string `json:"ip"`
	Labels struct {
		Cage string `json:"cage"`
	} `json:"labels"`
}

type Farm struct {
	Name  string `json:"name"`
	IP    string `json:"ip"`
	Cages []Cage `json:"cages"`
}

func main() {
	endpoint := "http://10.250.0.1:8181/"
	outputDir := filepath.Join(os.Getenv("HOME"), ".ssh", "prod")

	// Fetch data from the endpoint
	resp, err := http.Get(endpoint)
	if err != nil {
		fmt.Printf("Failed to fetch data from endpoint: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Non-OK HTTP status: %d\n", resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response body: %v\n", err)
		return
	}

	// Parse JSON data
	var farms []Farm
	if err := json.Unmarshal(body, &farms); err != nil {
		fmt.Printf("Failed to parse JSON: %v\n", err)
		return
	}

	// Ensure the output directory exists
	if err := os.MkdirAll(outputDir, 0700); err != nil {
		fmt.Printf("Failed to create output directory: %v\n", err)
		return
	}

	// Generate SSH config files
	for _, farm := range farms {
		generateConfigFile(outputDir, farm)
	}

	fmt.Println("SSH config files have been generated successfully.")
}

func generateConfigFile(outputDir string, farm Farm) {
	fileName := strings.ReplaceAll(farm.Name, "-", "_") + ".config"
	filePath := filepath.Join(outputDir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Failed to create config file %s: %v\n", filePath, err)
		return
	}
	defer file.Close()

	// Write farm host configuration
	farmAlias := strings.ReplaceAll(farm.Name, "-", "-") + "-farm"
	file.WriteString(fmt.Sprintf("Host %s\n", farmAlias))
	file.WriteString(fmt.Sprintf("    HostName %s\n\n", farm.IP))

	// Write cage configurations
	if len(farm.Cages) > 0 {
    file.WriteString(fmt.Sprintf("Host %s-cage-*\n", farmAlias))
		file.WriteString(fmt.Sprintf("    ProxyJump %s\n\n", farmAlias))

		for _, cage := range farm.Cages {
			cageAlias := fmt.Sprintf("%s-cage-%s", farmAlias, strings.ToLower(cage.Labels.Cage))
			file.WriteString(fmt.Sprintf("Host %s\n", cageAlias))
			file.WriteString(fmt.Sprintf("    HostName %s\n\n", cage.IP))
		}
	}
}

