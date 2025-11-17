package datastore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
)

// DatasetteClient implements the Store interface for remote Datasette instances
type DatasetteClient struct {
	baseURL  string
	apiToken string
	client   *http.Client
}

// NewDatasetteClient creates a new DatasetteClient instance
func NewDatasetteClient(baseURL, apiToken string) *DatasetteClient {
	return &DatasetteClient{
		baseURL:  baseURL,
		apiToken: apiToken,
		client:   &http.Client{},
	}
}

// Connect verifies the connection to the Datasette instance
func (c *DatasetteClient) Connect() error {
	// Parse and validate the base URL
	_, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	return nil
}

// CreateTable is a no-op for remote Datasette as tables are created via the insert API
func (c *DatasetteClient) CreateTable(schema string) error {
	// Tables are automatically created by the datasette-insert plugin
	return nil
}

// BatchInsert sends records to the Datasette insert API
func (c *DatasetteClient) BatchInsert(database string, table string, records []map[string]any) error {
	if len(records) == 0 {
		return nil
	}

	// Construct the API endpoint URL
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = path.Join(u.Path, "-/insert/hermes", table)

	// Prepare the request payload
	payload := map[string]any{
		"rows": records,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	if c.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	}

	// Send the request
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		var errResp map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("request failed with status %d", resp.StatusCode)
		}
		return fmt.Errorf("API error: %v", errResp)
	}

	return nil
}

// Close is a no-op for the HTTP client
func (c *DatasetteClient) Close() error {
	return nil
}
