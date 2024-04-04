package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"

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

		// Convert block height to string for GraphQL query
		fromBlockHeight := 1
		toBlockHeight := blockHeightQuery.LatestBlockHeight

		// Create the variables map
		variables := map[string]interface{}{
			"fromBlockHeight": fromBlockHeight,
			"toBlockHeight":   toBlockHeight,
		}

		// Create the GraphQL query
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

		// Create the request body
		requestBody := map[string]interface{}{
			"query":     query,
			"variables": variables,
		}

		// Convert the request body to JSON
		requestBodyJSON, err := json.Marshal(requestBody)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create JSON request"})
			return
		}

		// Create and send the HTTP request
		req, err := http.NewRequest("POST", *jsonrpcURL, bytes.NewBuffer(requestBodyJSON))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create HTTP request"})
			return
		}
		req.Header.Set("Content-Type", "application/json")

		// Send the HTTP request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send HTTP request"})
			return
		}
		defer resp.Body.Close()

		// Decode the response
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

		tmpl, err := template.New("dashboard").Parse(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>Dashboard</title>
			</head>
			<body>
				<h1>Dashboard</h1>
				<p>Current Block Height: {{ .LatestBlockHeight }}</p>
				<p>Total Transactions Since Block 1: {{ .TotalTransactionsSinceBlock1 }}</p>
			</body>
			</html>
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse template"})
			return
		}

		var tplBuffer bytes.Buffer
		data := gin.H{
			"LatestBlockHeight":            blockHeightQuery.LatestBlockHeight,
			"TotalTransactionsSinceBlock1": totalTransactions,
		}
		if err := tmpl.Execute(&tplBuffer, data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute template"})
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", tplBuffer.Bytes())
	})

	if err := router.Run(":8080"); err != nil {
		log.Fatal("Error starting server:", err)
	}
}
