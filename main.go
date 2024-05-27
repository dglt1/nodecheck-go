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
	urls := readURLs("nodes.txt")
	if len(urls) < 2 {
		log.Fatal("Not enough URLs in the file")
	}

	url1, url2 := urls[0], urls[1]

	for {
		clearScreen()

		response1 := makeRequest(url1)
		response2 := makeRequest(url2)

		slot1, blockHeight1 := parseValues(response1)
		slot2, blockHeight2 := parseValues(response2)

		slotDiff := slot1 - slot2

		fmt.Printf("Response from %s\nSlot: %d, BlockHeight: %d\n", url1, slot1, blockHeight1)
		fmt.Printf("Response from %s\nSlot: %d, BlockHeight: %d\n", url2, slot2, blockHeight2)
		fmt.Printf("\n%s is %d slots behind\n", url2, slotDiff)

		time.Sleep(3 * time.Second)
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
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(payload))
	if err != nil {
		log.Fatalf("HTTP request failed: %s", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %s", err)
	}
	return body
}

func parseValues(jsonData []byte) (int, int) {
	var result []map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		log.Fatalf("JSON unmarshalling failed: %s", err)
	}

	slot := 0
	blockHeight := 0

	// Check for slot
	if val, ok := result[1]["result"].(float64); ok {
		slot = int(val)
	} else if val, ok := result[1]["result"].(int); ok {
		slot = val
	} else {
		log.Fatalf("Expected numeric type for slot but got different type")
	}

	if val, ok := result[2]["result"].(float64); ok {
		blockHeight = int(val)
	} else if val, ok := result[2]["result"].(int); ok {
		blockHeight = val
	} else {
		log.Fatalf("Expected numeric type for block height but got different type")
	}

	return slot, blockHeight
}
