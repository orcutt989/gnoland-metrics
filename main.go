package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shurcooL/graphql"
)

var (
	jsonrpcURL = flag.String("jsonrpc-url", "", "URL of the JSON RPC API")
)

// BlockHeightResponse represents the response containing the latest block height.
type BlockHeightResponse struct {
	LatestBlockHeight int `json:"latestBlockHeight"`
}

// Transaction represents a blockchain transaction.
type Transaction struct {
	Index       int    `json:"index"`
	Hash        string `json:"hash"`
	BlockHeight int    `json:"block_height"`
	GasWanted   int    `json:"gas_wanted"`
	GasUsed     int    `json:"gas_used"`
	ContentRaw  string `json:"content_raw"`
}

// Block represents a block in the blockchain.
type Block struct {
	Height int       `json:"height"`
	Time   time.Time `json:"time"`
}

func main() {
	flag.Parse()

	if *jsonrpcURL == "" {
		log.Fatal("Error: jsonrpc-url flag is required")
	}

	router := gin.Default()

	// Handle GET request for /dashboard endpoint
	router.GET("/dashboard", func(c *gin.Context) {
		client := graphql.NewClient(*jsonrpcURL, nil)

		// Fetch latest block height
		var blockHeightQuery struct {
			LatestBlockHeight int `graphql:"latestBlockHeight"`
		}
		if err := client.Query(context.Background(), &blockHeightQuery, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch block height"})
			return
		}

		// Convert block height to string for GraphQL query
		fromBlockHeight := 1
		toBlockHeight := blockHeightQuery.LatestBlockHeight

		// Create the variables map for GraphQL query
		variables := map[string]interface{}{
			"fromBlockHeight": fromBlockHeight,
			"toBlockHeight":   toBlockHeight,
		}

		// Create the GraphQL query for fetching transactions
		query := `
			query TotalTransactions($fromBlockHeight: Int, $toBlockHeight: Int) {
				transactions(filter: { from_block_height: $fromBlockHeight, to_block_height: $toBlockHeight }) {
					index
					hash
					block_height
					gas_wanted
					gas_used
					content_raw
				}
			}
		`

		// Execute the GraphQL query to fetch transactions
		resp, err := executeGraphQLQuery(client, variables, query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
			return
		}
		defer resp.Body.Close()

		// Decode the response to get total transactions
		var totalTransactionsQuery struct {
			Data struct {
				Transactions []Transaction `json:"transactions"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&totalTransactionsQuery); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode JSON response"})
			return
		}

		totalTransactions := len(totalTransactionsQuery.Data.Transactions)

		// Get the time 6 hours ago
		sixHoursAgo := time.Now().Add(-6 * time.Hour)

		// Create the variables map for GraphQL query to fetch blocks
		variablesBlocks := map[string]interface{}{
			"fromTime": sixHoursAgo.Format(time.RFC3339),
			"toTime":   time.Now().Format(time.RFC3339),
		}

		// Create the GraphQL query for fetching blocks
		blockQuery := `
			query BlocksWithinTimeRange($fromTime: Time!, $toTime: Time!) {
				blocks(filter: { from_time: $fromTime, to_time: $toTime }) {
					height
					time
				}
			}
		`

		// Execute the GraphQL query to fetch blocks
		blockResp, err := executeGraphQLQuery(client, variablesBlocks, blockQuery)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch blocks"})
			return
		}
		defer blockResp.Body.Close()

		// Decode the response to get blocks within the time range
		var blocksWithinTime struct {
			Data struct {
				Blocks []Block `json:"blocks"`
			} `json:"data"`
		}
		if err := json.NewDecoder(blockResp.Body).Decode(&blocksWithinTime); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode JSON response for blocks"})
			return
		}

		// Calculate transactions per hour based on blocks within the time range
		transactionsPerHour := make(map[string]int)
		for _, block := range blocksWithinTime.Data.Blocks {
			hour := block.Time.Format("2006-01-02 15:00:00") // Round time to the hour
			transactionsPerHour[hour]++
		}

		// Prepare data for HTML template
		data := gin.H{
			"TransactionsPerHour":          transactionsPerHour,
			"LatestBlockHeight":            blockHeightQuery.LatestBlockHeight,
			"TotalTransactionsSinceBlock1": totalTransactions,
		}

		// Render HTML template and send response
		renderHTMLTemplate(c, data)
	})

	// Start the HTTP server
	if err := router.Run(":8080"); err != nil {
		log.Fatal("Error starting server:", err)
	}
}

// executeGraphQLQuery executes a GraphQL query using the provided client, variables, and query string.
func executeGraphQLQuery(client *graphql.Client, variables map[string]interface{}, query string) (*http.Response, error) {
	// Create the request body
	requestBody := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	// Convert the request body to JSON
	requestBodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	// Create and send the HTTP request
	req, err := http.NewRequest("POST", *jsonrpcURL, bytes.NewBuffer(requestBodyJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Check the HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("non-200 OK status code: " + resp.Status)
	}

	return resp, nil
}

// renderHTMLTemplate renders the HTML template with the provided data and sends the response.
func renderHTMLTemplate(c *gin.Context, data gin.H) {
	tmpl, err := template.New("dashboard").Parse(`
    <!DOCTYPE html>
    <html>
    <head>
        <title>Dashboard</title>
        <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    </head>
    <body>
        <h1>Dashboard</h1>
        <p>Current Block Height: {{ .LatestBlockHeight }}</p>
        <p>Total Transactions Since Block 1: {{ .TotalTransactionsSinceBlock1 }}</p>
        <canvas id="transactionsChart" width="800" height="400"></canvas>
        <script>
            var ctx = document.getElementById('transactionsChart').getContext('2d');
    
            var myChart = new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: [{{ range $key, $value := .TransactionsPerHour }}"{{$key}}",{{ end }}],
                    datasets: [{
                        label: 'Transactions per Hour',
                        data: [{{ range $key, $value := .TransactionsPerHour }}{{$value}},{{ end }}],
                        backgroundColor: 'rgba(54, 162, 235, 0.5)',
                        borderColor: 'rgba(54, 162, 235, 1)',
                        borderWidth: 1
                    }]
                },
                options: {
                    scales: {
                        y: {
                            beginAtZero: true
                        }
                    }
                }
            });
        </script>
    </body>
    </html>
    
	`)
	if err != nil {
		log.Printf("Failed to parse template: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse template"})
		return
	}

	var tplBuffer bytes.Buffer
	if err := tmpl.Execute(&tplBuffer, data); err != nil {
		log.Printf("Failed to execute template: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute template"})
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", tplBuffer.Bytes())
}
