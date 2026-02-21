package handlers

import (
	"customer-help-center-backend/internal/models"
	"customer-help-center-backend/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	service service.AdminService
}

func NewAdminHandler(service service.AdminService) *AdminHandler {
	return &AdminHandler{service: service}
}

func (h *AdminHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.RegisterAdmin(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// GetProfile handles fetching the authenticated user's profile (GET /profile)
// func (h *AuthHandler) GetProfile(c *gin.Context) {
// 	// The UID is set in the AuthMiddleware
// 	uid, exists := c.Get("uid")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
// 		return
// 	}

// 	user, err := h.authService.GetProfileByUID(uid.(string))
// 	if err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "User profile not found"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, user)
// }

// UpdateProfile handles updating the authenticated user's profile (PUT /profile)
func (h *AdminHandler) UpdateProfile(c *gin.Context) {
	uid, exists := c.Get("uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.UpdateProfile(uid.(string), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully", "user": user})
}
func (h *AdminHandler) GetProfile(c *gin.Context) {
	// Example: get ID from param or auth context
	uid, exists := c.Get("uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	admin, err := h.service.GetProfileByUID(uid.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}

	c.JSON(http.StatusOK, admin)
}

func (h *AdminHandler) ListAdmins(c *gin.Context) {
	admins, err := h.service.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, admins)
}

func (h *AdminHandler) DeleteAdmin(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UID required"})
		return
	}

	if err := h.service.DeleteAdmin(uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Admin deleted successfully"})
}

func (h *AdminHandler) UpdateAdmin(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UID required"})
		return
	}

	var req models.Admin
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedAdmin, err := h.service.UpdateAdmin(uid, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedAdmin)
}
func (h *AdminHandler) Login(c *gin.Context) {
	var req models.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	maxAge := 3600 * 24

	// Set cookie with secure=false for development (should be configurable for prod)
	// SameSite defaults to Lax/Strict behavior in modern browsers when not specified with Secure
	c.SetCookie(
		"admin_token", // name
		token,         // value
		maxAge,        // maxAge in seconds
		"/",           // path
		"",            // domain
		false,         // secure: false allows http on localhost
		true,          // httpOnly
	)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"uid":   user.UID,
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

func (h *AdminHandler) GetAgentPerformance(c *gin.Context) {
	performance, err := h.service.GetAgentPerformance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, performance)
}

func (h *AdminHandler) Logout(c *gin.Context) {
	c.SetCookie("admin_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
