package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
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

type BlockHeightResponse struct {
	LatestBlockHeight int `json:"latestBlockHeight"`
}

type Transaction struct {
	Index       int    `json:"index"`
	Hash        string `json:"hash"`
	BlockHeight int    `json:"block_height"`
	GasWanted   int    `json:"gas_wanted"`
	GasUsed     int    `json:"gas_used"`
	ContentRaw  string `json:"content_raw"`
}

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

		// Get the time 6 hours ago
		sixHoursAgo := time.Now().Add(-6 * time.Hour)

		// Create the variables map for fetching blocks
		variables := map[string]interface{}{
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

		// Create the request body for fetching blocks
		blockRequestBody := map[string]interface{}{
			"query":     blockQuery,
			"variables": variables,
		}

		// Convert the block request body to JSON
		blockRequestBodyJSON, err := json.Marshal(blockRequestBody)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create JSON request for blocks"})
			return
		}

		// Create and send the HTTP request to fetch blocks
		blockReq, err := http.NewRequest("POST", *jsonrpcURL, bytes.NewBuffer(blockRequestBodyJSON))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create HTTP request for blocks"})
			return
		}
		blockReq.Header.Set("Content-Type", "application/json")

		// Send the HTTP request to fetch blocks
		blockResp, err := http.DefaultClient.Do(blockReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send HTTP request for blocks"})
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
			"TransactionsPerHour": transactionsPerHour,
			"LatestBlockHeight":   blockHeightQuery.LatestBlockHeight,
		}

		tmpl, err := template.New("dashboard").Parse(`
    <!DOCTYPE html>
    <html>
    <head>
        <title>Dashboard</title>
        <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    </head>
    <body>
        <h1>Dashboard</h1>
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
        <p>Current Block Height: {{ .LatestBlockHeight }}</p>
    </body>
    </html>
    
		`)
		if err != nil {
			log.Printf("Failed to parse template: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse template"})
			return
		}

		// Debugging: Print the template and data
		fmt.Println("HTML Template:")
		fmt.Println(tmpl)

		fmt.Println("Data:")
		fmt.Println(data)

		var tplBuffer bytes.Buffer
		if err := tmpl.Execute(&tplBuffer, data); err != nil {
			log.Printf("Failed to execute template: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute template"})
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", tplBuffer.Bytes())
	})

	if err := router.Run(":8080"); err != nil {
		log.Fatal("Error starting server:", err)
	}
}
