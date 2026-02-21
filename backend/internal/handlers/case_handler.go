package handlers

import (
	"net/http"

	"customer-help-center-backend/internal/models"
	"customer-help-center-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type CaseHandler struct {
	service service.CaseService
}

func NewCaseHandler(s service.CaseService) *CaseHandler {
	return &CaseHandler{service: s}
}

// CreateCase godoc
// @Summary Create a new case
// @Description Create a new case
// @Tags cases
// @Accept json
// @Produce json
// @Param case body models.Case true "Case"
// @Success 200 {object} models.Case
// @Router /cases [post]
func (h *CaseHandler) CreateCase(c *gin.Context) {
	var req models.Case
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newCase, err := h.service.CreateCase(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create case"})
		return
	}

	c.JSON(http.StatusOK, newCase)
}

// GetCases godoc
// @Summary Get all cases
// @Description Get all cases
// @Tags cases
// @Produce json
// @Success 200 {array} models.Case
// @Router /cases [get]
func (h *CaseHandler) GetCases(c *gin.Context) {
	cases, err := h.service.GetAllCases()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cases"})
		return
	}

	c.JSON(http.StatusOK, cases)
}

// GetCase godoc
// @Summary Get a case by ID
// @Description Get a case by ID
// @Tags cases
// @Produce json
// @Param id path int true "Case ID"
// @Success 200 {object} models.Case
// @Router /cases/{id} [get]
func (h *CaseHandler) GetCase(c *gin.Context) {
	var id struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.GetCase(id.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
		return
	}

	c.JSON(http.StatusOK, res)
}

// UpdateCase godoc
// @Summary Update a case
// @Description Update a case
// @Tags cases
// @Accept json
// @Produce json
// @Param id path int true "Case ID"
// @Param case body models.Case true "Case"
// @Success 200 {object} models.Case
// @Router /cases/{id} [put]
func (h *CaseHandler) UpdateCase(c *gin.Context) {
	var id struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req models.Case
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.service.UpdateCase(id.ID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update case"})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteCase godoc
// @Summary Delete a case
// @Description Delete a case
// @Tags cases
// @Param id path int true "Case ID"
// @Success 200 {object} map[string]string
// @Router /cases/{id} [delete]
func (h *CaseHandler) DeleteCase(c *gin.Context) {
	var id struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.DeleteCase(id.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete case"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Case deleted successfully"})
}
