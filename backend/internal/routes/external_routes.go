package routes

import (
	"customer-help-center-backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

func SetupExternalRoutes(r *gin.Engine, externalH *handlers.ExternalHandler) {
	// Public routes for external integrations
	external := r.Group("/api/external")
	{
		external.GET("/receipts/:id", externalH.ExternalReceiptLookup)
	}
}
