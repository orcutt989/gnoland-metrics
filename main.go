package main

import (
	"bytes"
	"context"
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

func main() {
	flag.Parse()

	if *jsonrpcURL == "" {
		log.Fatal("Error: jsonrpc-url flag is required")
	}

	router := gin.Default()

	router.GET("/dashboard", func(c *gin.Context) {
		client := graphql.NewClient(*jsonrpcURL, nil)

		var query struct {
			LatestBlockHeight int `graphql:"latestBlockHeight"`
		}

		if err := client.Query(context.Background(), &query, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch block height"})
			return
		}

		tmpl, err := template.New("dashboard").Parse(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>Dashboard</title>
			</head>
			<body>
				<h1>Dashboard</h1>
				<p>Current Block Height: {{ .LatestBlockHeight }}</p>
			</body>
			</html>
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse template"})
			return
		}

		var tplBuffer bytes.Buffer
		data := gin.H{"LatestBlockHeight": query.LatestBlockHeight}
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
