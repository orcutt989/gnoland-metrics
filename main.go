package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"

	"github.com/gorilla/websocket"
)

type GraphQLRequest struct {
	Query string `json:"query"`
}

type GraphQLResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

func executeGraphQLQuery(wsURL, query string) (*GraphQLResponse, error) {
	u := url.URL{Scheme: "ws", Host: wsURL, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	requestBody, err := json.Marshal(GraphQLRequest{Query: query})
	if err != nil {
		return nil, err
	}

	if err := c.WriteMessage(websocket.TextMessage, requestBody); err != nil {
		return nil, err
	}

	var response GraphQLResponse
	_, message, err := c.ReadMessage()
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(message, &response); err != nil {
		return nil, err
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", response.Errors[0].Message)
	}

	return &response, nil
}

func main() {
	var listenAddress string
	flag.StringVar(&listenAddress, "listen-address", "localhost:8545", "WebSocket endpoint address")
	flag.Parse()

	wsURL := listenAddress

	// GraphQL queries
	queries := map[string]string{
		"transactionsCount": `
			query {
				blocks(filter: { from_height: 1, to_height: 999999 }) {
					height
					transactions {
						index
					}
				}
			}
		`,
		"distinctMessageTypesCount": `
			query {
				transactions(filter: { from_block_height: 1 }) {
					hash
				}
			}
		`,
		"mostActiveTransactionSenders": `
			query {
				transactions(filter: { from_block_height: 1 }) {
					index
					gas_wanted
				}
			}
		`,
	}

	for metricName, query := range queries {
		response, err := executeGraphQLQuery(wsURL, query)
		if err != nil {
			log.Fatalf("Error executing GraphQL query for %s: %v", metricName, err)
		}

		if len(response.Errors) > 0 {
			for _, e := range response.Errors {
				log.Printf("Error for %s: %s", metricName, e.Message)
			}
		} else {
			fmt.Printf("Metric: %s\n", metricName)
			fmt.Printf("Response: %+v\n", response.Data)
			fmt.Println("----------------------------------------")
		}
	}
}
