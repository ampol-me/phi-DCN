package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"phi-DCN/client/config"
)

// SSEEvent represents a single SSE event
type SSEEvent struct {
	Data []byte
}

// SSEClient handles SSE connection
type SSEClient struct {
	client   *http.Client
	url      string
	events   chan SSEEvent
	stopChan chan struct{}
}

// NewSSEClient creates a new SSE client
func NewSSEClient() *SSEClient {
	return &SSEClient{
		client: &http.Client{
			Timeout: 0, // No timeout for SSE
		},
		url:      "http://" + config.Config.APIHost + ":" + config.Config.APIPort + "/api/sse",
		events:   make(chan SSEEvent, 100),
		stopChan: make(chan struct{}),
	}
}

// Connect establishes SSE connection
func (c *SSEClient) Connect() error {
	req, err := http.NewRequest("GET", c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set SSE headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	// Start reading events in a goroutine
	go c.readEvents(resp.Body)

	return nil
}

// readEvents reads SSE events from the response body
func (c *SSEClient) readEvents(body io.ReadCloser) {
	defer body.Close()
	reader := bufio.NewReader(body)

	for {
		select {
		case <-c.stopChan:
			return
		default:
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					fmt.Printf("Error reading SSE: %v\n", err)
				}
				return
			}

			// Skip empty lines and comments
			if len(line) <= 2 || bytes.HasPrefix(line, []byte(":")) {
				continue
			}

			// Process data line
			if bytes.HasPrefix(line, []byte("data: ")) {
				data := bytes.TrimSpace(line[6:])
				c.events <- SSEEvent{Data: data}
			}
		}
	}
}

// GetEvents returns the events channel
func (c *SSEClient) GetEvents() <-chan SSEEvent {
	return c.events
}

// Stop closes the SSE connection
func (c *SSEClient) Stop() {
	close(c.stopChan)
}

// ProcessSSEEvents processes SSE events and updates speaker data
func ProcessSSEEvents() {
	sseClient := NewSSEClient()
	if err := sseClient.Connect(); err != nil {
		fmt.Printf("Failed to connect to SSE: %v\n", err)
		return
	}
	defer sseClient.Stop()

	for event := range sseClient.GetEvents() {
		var speakers []Speaker
		if err := json.Unmarshal(event.Data, &speakers); err != nil {
			fmt.Printf("Failed to parse SSE data: %v\n", err)
			continue
		}

		// Update active mics
		for _, speaker := range speakers {
			config.Config.UpdateActiveMic(speaker.Name, speaker.MicOn == 1)
		}

		// Broadcast to TCP clients
		// TODO: Implement broadcasting to TCP clients
	}
}
