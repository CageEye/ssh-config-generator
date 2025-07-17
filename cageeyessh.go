package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Cage struct {
	IP     string `json:"ip"`
	Labels struct {
		Cage                 string `json:"cage"`
		CageProcessingUnitId string `json:"cage_processing_unit_id"`
	} `json:"labels"`
}

type Farm struct {
	Name  string `json:"name"`
	IP    string `json:"ip"`
	Cages []Cage `json:"cages"`
}

func main() {
	prodEndpoint := "http://10.250.0.1:8181/"
	stageEndpoint := "http://10.240.0.1:8181/"

	// Create an HTTP client with a timeout of 2 seconds
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Determine which VPN connections are active
	vpnTypes, err := checkVPNConnection()
	if err != nil {
		fmt.Printf("Error checking VPN connection: %v\n", err)
		return
	}

	if len(vpnTypes) == 0 {
		fmt.Println("No active VPN connection found (prod or stage).")
		return
	}

	var includeLines []string
	for _, vpnType := range vpnTypes {
		var endpoint, outputDir, configIncludeLine string
		switch vpnType {
		case "prod":
			endpoint = prodEndpoint
			outputDir = filepath.Join(os.Getenv("HOME"), ".ssh", "prod")
			configIncludeLine = "Include prod.config"
			fmt.Println("Prod VPN connection detected.")
		case "stage":
			endpoint = stageEndpoint
			outputDir = filepath.Join(os.Getenv("HOME"), ".ssh", "staging")
			configIncludeLine = "Include staging.config"
			fmt.Println("Stage VPN connection detected.")
		default:
			continue
		}

		// Fetch data from the determined endpoint
		resp, err := client.Get(endpoint)
		if err != nil {
			fmt.Printf("Failed to fetch data from endpoint %s: %v\n", endpoint, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Non-OK HTTP status from %s: %d\n", endpoint, resp.StatusCode)
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Failed to read response body: %v\n", err)
			continue
		}

		// Parse JSON data
		var farms []Farm
		if err := json.Unmarshal(body, &farms); err != nil {
			fmt.Printf("Failed to parse JSON: %v\n", err)
			continue
		}

		// Ensure the output directory exists
		if err := os.MkdirAll(outputDir, 0700); err != nil {
			fmt.Printf("Failed to create output directory: %v\n", err)
			continue
		}

		// Generate SSH config files
		for _, farm := range farms {
			generateConfigFile(outputDir, farm)
		}

		includeLines = append(includeLines, configIncludeLine)
		fmt.Printf("SSH config files have been generated successfully in %s.\n", outputDir)
	}

	updateSSHConfig(includeLines)

	fmt.Printf("Main SSH config updated with: %s\n", strings.Join(includeLines, ", "))
}

func checkVPNConnection() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var vpnTypes []string
	for _, i := range interfaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil {
				if strings.HasPrefix(ip.String(), "10.251.") {
					vpnTypes = append(vpnTypes, "prod")
				}
				if strings.HasPrefix(ip.String(), "10.241.") {
					vpnTypes = append(vpnTypes, "stage")
				}
			}
		}
	}

	return vpnTypes, nil
}

func formatCageLabel(label string) string {
	// Remove any leading/trailing spaces
	label = strings.TrimSpace(label)
	// Replace spaces with hyphens and convert to lowercase
	label = strings.ReplaceAll(label, " ", "-")
	return strings.ToLower(label)
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
			var cageAlias string
			if cage.Labels.Cage == "" {
				macParts := strings.Split(cage.Labels.CageProcessingUnitId, ":")
				cageID := strings.Join(macParts[len(macParts)-3:], "-")
				cageAlias = fmt.Sprintf("%s-cage-%s", farmAlias, strings.ToLower(cageID))
			} else {
				formattedCageLabel := formatCageLabel(cage.Labels.Cage)
				cageAlias = fmt.Sprintf("%s-cage-%s", farmAlias, formattedCageLabel)
			}
			file.WriteString(fmt.Sprintf("Host %s\n", cageAlias))
			file.WriteString(fmt.Sprintf("    HostName %s\n\n", cage.IP))
		}
	}
}

func updateSSHConfig(includeLines []string) {
	configPath := filepath.Join(os.Getenv("HOME"), ".ssh", "config")

	// Read the current config file
	content, err := ioutil.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		fmt.Printf("Error reading SSH config file: %v\n", err)
		return
	}

	// Create a new file if it doesn	 exist
	if os.IsNotExist(err) {
		if err := ioutil.WriteFile(configPath, []byte(strings.Join(includeLines, "\n")+"\n"), 0600); err != nil {
			fmt.Printf("Error creating SSH config file: %v\n", err)
		}
		return
	}

	// Find and remove old Include lines
	lines := strings.Split(string(content), "\n")
	var newLines []string
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "Include prod.config") && !strings.HasPrefix(strings.TrimSpace(line), "Include staging.config") {
			newLines = append(newLines, line)
		}
	}

	// Add the new Include lines to the beginning
	newLines = append(includeLines, newLines...)

	// Write the updated config back to file
	newContent := strings.Join(newLines, "\n")
	if err := ioutil.WriteFile(configPath, []byte(newContent), 0600); err != nil {
		fmt.Printf("Error updating SSH config file: %v\n", err)
	}
}
