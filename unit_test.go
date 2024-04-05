package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecuteGraphQLQuery(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Define the response body JSON
		responseBody := `{"data": {"transactions": [{"index": 1, "hash": "abc123", "block_height": 123, "gas_wanted": 1000, "gas_used": 800, "content_raw": "Lorem ipsum"}]}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseBody))
	}))
	defer server.Close()

	// Create a mock JSON request body
	requestBody := map[string]interface{}{
		"query": "mock query",
		"variables": map[string]interface{}{
			"mockVar": "mockValue",
		},
	}

	// Convert the request body to JSON
	requestBodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("Failed to marshal JSON request body: %v", err)
	}

	// Create and send the HTTP request to the mock server
	req, err := http.NewRequest("POST", server.URL, bytes.NewBuffer(requestBodyJSON))
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Decode the response
	var data struct {
		Data struct {
			Transactions []Transaction `json:"transactions"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	// Assert that the response data is as expected
	expected := []Transaction{{Index: 1, Hash: "abc123", BlockHeight: 123, GasWanted: 1000, GasUsed: 800, ContentRaw: "Lorem ipsum"}}
	assert.Equal(t, expected, data.Data.Transactions, "Response data doesn't match expected")
}
