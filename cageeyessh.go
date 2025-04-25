package main

import (
        "encoding/json"
        "fmt"
        "io/ioutil"
        "net/http"
        "os"
        "time"
        "path/filepath"
        "strings"
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


        // Fetch data from the prodEndpoint
        resp, err := client.Get(prodEndpoint)
        endpoint := prodEndpoint
        outputDir := filepath.Join(os.Getenv("HOME"), ".ssh", "prod")
        configIncludeLine := "Include prod.config"

        if err != nil || resp.StatusCode != http.StatusOK {
            if resp != nil {
                resp.Body.Close()
            }
            fmt.Println("Prod endpoint not available, trying staging endpoint...")
            resp, err = client.Get(stageEndpoint)
            endpoint = stageEndpoint
            outputDir = filepath.Join(os.Getenv("HOME"), ".ssh", "staging")
            configIncludeLine = "Include staging.config"
        }

        // Check if the request was successful
        if err != nil {
            fmt.Printf("Failed to fetch data from endpoints: %v\n", err)
            return
        }

        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
                fmt.Printf("Non-OK HTTP status from %s: %d\n", endpoint, resp.StatusCode)
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

        updateSSHConfig(configIncludeLine)

        fmt.Printf("SSH config files have been generated successfully in %s.\n", outputDir)
        fmt.Printf("Main SSH config updated with: %s\n", configIncludeLine)
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

func updateSSHConfig(includeLine string) {
    configPath := filepath.Join(os.Getenv("HOME"), ".ssh", "config")

    // Read the current config file
    content, err := ioutil.ReadFile(configPath)
    if err != nil && !os.IsNotExist(err) {
        fmt.Printf("Error reading SSH config file: %v\n", err)
        return
    }

    // Create a new file if it doesn't exist
    if os.IsNotExist(err) {
        if err := ioutil.WriteFile(configPath, []byte(includeLine+"\n"), 0600); err != nil {
            fmt.Printf("Error creating SSH config file: %v\n", err)
        }
        return
    }

    // Find and replace the Include line if it exists
    lines := strings.Split(string(content), "\n")
    includeFound := false

    for i, line := range lines {
        if strings.HasPrefix(strings.TrimSpace(line), "Include prod.config") {
            lines[i] = includeLine
            includeFound = true
            break
        }
        if strings.HasPrefix(strings.TrimSpace(line), "Include staging.config") {
            lines[i] = includeLine
            includeFound = true
            break
        }
    }

    // If no Include line was found, add it to the beginning
    if !includeFound {
        lines = append([]string{includeLine}, lines...)
    }

    // Write the updated config back to file
    newContent := strings.Join(lines, "\n")
    if err := ioutil.WriteFile(configPath, []byte(newContent), 0600); err != nil {
        fmt.Printf("Error updating SSH config file: %v\n", err)
    }
}

