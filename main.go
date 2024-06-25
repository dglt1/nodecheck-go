package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func main() {
	defaultURL := "https://api.mainnet-beta.solana.com"
	urls := readURLs("nodes.txt")

	for {
		clearScreen()

		defaultResponse := makeRequest(defaultURL)
		defaultSlot, defaultBlockHeight := parseValues(defaultResponse)

		fmt.Printf("(%s) Slot: %d, BlockHeight: %d\n", defaultURL, defaultSlot, defaultBlockHeight)
		fmt.Println()

		for _, url := range urls {
			response := makeRequest(url)
			if response == nil {
				log.Printf("No valid response from %s, skipping.", url)
				continue
			}
			slot, blockHeight := parseValues(response)
			if slot == 0 && blockHeight == 0 {
				log.Printf("Invalid data from %s, skipping.", url)
				continue
			}
			slotDiff := defaultSlot - slot

			fmt.Printf("Response from %s\nSlot: %d, BlockHeight: %d\n", url, slot, blockHeight)
			fmt.Printf("%s is %d slots behind mainnet\n\n", url, slotDiff)

			// Append to behind.log if slotDiff is more than 4
			if slotDiff > 4 {
				logToFile(url, slotDiff)
			}
		}

		time.Sleep(3 * time.Second)
	}
}

func logToFile(url string, slotDiff int) {
	f, err := os.OpenFile("behind.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open log file: %s", err)
		return
	}
	defer f.Close()

	logEntry := fmt.Sprintf("%s is %d slots behind mainnet\n", url, slotDiff)
	if _, err := f.WriteString(logEntry); err != nil {
		log.Printf("Failed to write to log file: %s", err)
	}
}

func readURLs(filePath string) []string {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %s", err)
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read file: %s", err)
	}
	return urls
}

func makeRequest(url string) []byte {
	payload := `[{"jsonrpc":"2.0","id":1, "method":"getHealth"},{"jsonrpc":"2.0","id":2, "method":"getSlot"},{"jsonrpc":"2.0","id":3, "method":"getBlockHeight"}]`
	client := &http.Client{
		Timeout: 2 * time.Second, // Set timeout for the HTTP client
	}
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(payload))
	if err != nil {
		log.Printf("Failed to create request for %s: %s", url, err)
		return nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP request to %s failed: %s", url, err)
		return nil // Return nil to indicate a failed or timed-out request
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Non-OK HTTP status from %s: %d", url, resp.StatusCode)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body from %s: %s", url, err)
		return nil
	}
	return body
}

func parseValues(jsonData []byte) (int, int) {
	if jsonData == nil {
		return 0, 0 // Return zero values if jsonData is nil
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		log.Printf("JSON unmarshalling failed: %s", err)
		return 0, 0 // Return zero values if unmarshalling fails
	}

	slot := 0
	blockHeight := 0

	// Extract slot and block height safely
	if val, ok := result[1]["result"].(float64); ok {
		slot = int(val)
	} else if val, ok := result[1]["result"].(int); ok {
		slot = val
	}

	if val, ok := result[2]["result"].(float64); ok {
		blockHeight = int(val)
	} else if val, ok := result[2]["result"].(int); ok {
		blockHeight = val
	}

	return slot, blockHeight
}
