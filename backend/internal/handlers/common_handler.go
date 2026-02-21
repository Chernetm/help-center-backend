package handlers

import (
	"customer-help-center-backend/internal/models"
	"customer-help-center-backend/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CommonHandler struct {
	service service.CommonService
}

func NewCommonHandler(s service.CommonService) *CommonHandler {
	return &CommonHandler{service: s}
}

func (h *CommonHandler) SubmitRating(c *gin.Context) {
	var rating models.Rating
	if err := c.ShouldBindJSON(&rating); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if err := h.service.SubmitRating(&rating); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rating)
}
