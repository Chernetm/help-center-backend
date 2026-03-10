package handlers

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ExternalHandler struct{}

func NewExternalHandler() *ExternalHandler {
	return &ExternalHandler{}
}

// ReceiptLookup fetches data from the external ngrok API
func (h *ExternalHandler) ExternalReceiptLookup(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "order_id is required"})
		return
	}

	url := "https://demetrius-unrepressive-bootlessly.ngrok-free.dev/api/orders/" + orderID

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create request"})
		return
	}
	// Bypass ngrok browser warning for free accounts
	req.Header.Set("ngrok-skip-browser-warning", "true")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": "failed to reach external receipt service"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, gin.H{
			"message": "external service returned error",
			"details": string(body),
		})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to read response body"})
		return
	}

	c.Data(http.StatusOK, "application/json", body)
}
