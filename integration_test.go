package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	router := gin.Default()

	// Set up the /dashboard endpoint
	router.GET("/dashboard", func(c *gin.Context) {
		// Simulate rendering HTML for dashboard
		htmlContent := `
			<!DOCTYPE html>
			<html>
			<head>
				<title>Dashboard</title>
				<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
			</head>
			<body>
				<h1>Dashboard</h1>
				<p>Current Block Height: 1000</p>
				<p>Total Transactions Since Block 1: 100</p>
				<canvas id="transactionsChart" width="800" height="400"></canvas>
				<script>
					var ctx = document.getElementById('transactionsChart').getContext('2d');
		
					var myChart = new Chart(ctx, {
						type: 'bar',
						data: {
							labels: [],
							datasets: [{
								label: 'Transactions per Hour',
								data: [],
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
		`
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
	})

	return router
}

func TestIntegrationDashboardEndpoint(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dashboard", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK; got %d", w.Code)
	}

	if body := w.Body.String(); !strings.Contains(body, "<html>") {
		t.Errorf("expected HTML body to contain \"<html>\"; got %q", body)
	}
}
