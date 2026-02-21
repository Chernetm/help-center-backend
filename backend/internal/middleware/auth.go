package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"customer-help-center-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// // AuthMiddleware is a Gin middleware to check for a valid Firebase ID Token
// func AuthMiddleware(authService service.AdminService) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		authHeader := c.GetHeader("Authorization")
// 		if authHeader == "" {
// 			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
// 			c.Abort()
// 			return
// 		}

// 		// Check if the header is in the format "Bearer <token>"
// 		parts := strings.Split(authHeader, " ")
// 		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
// 			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
// 			c.Abort()
// 			return
// 		}

// 		tokenString := parts[1]

// 		// Verify the Firebase ID Token
// 		token, err := authService.VerifyIDToken(tokenString)
// 		if err != nil {
// 			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
// 			c.Abort()
// 			return
// 		}

// 		// Set the user UID in the context for handlers to use
// 		c.Set("uid", token.UID)

// 		c.Next()
// 	}
// }

// FirebaseAuthMiddleware verifies the Firebase ID Token and sets the UID in the context.
// It does NOT check the local database for user existence.
func FirebaseAuthMiddleware(adminService service.AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		var tokenString string

		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				tokenString = parts[1]
			}
		}

		// Fallback to cookie
		if tokenString == "" {
			tokenString, _ = c.Cookie("admin_token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			c.Abort()
			return
		}

		token, err := adminService.VerifyIDToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("uid", token.UID)
		c.Set("claims", token.Claims)
		c.Next()
	}
}

// AdminMiddleware checks if the authenticated user exists in the Admin table.
// Requires FirebaseAuthMiddleware to be run first (or duplicates logic if standalone, here we assume standalone or chained).
// To keep it simple and independent as per current usage pattern, let's keep it verifying token THEN checking DB.
// But leveraging FirebaseAuthMiddleware logic is better.
// Let's implement full check here to be safe replacement.
func AdminMiddleware(adminService service.AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Verify Token (or get from context if chained)
		uid, exists := c.Get("uid")
		if !exists {
			// If not in context, verify header manually
			authHeader := c.GetHeader("Authorization")
			var tokenString string

			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					tokenString = parts[1]
				}
			}

			if tokenString == "" {
				tokenString, _ = c.Cookie("admin_token")
			}

			if tokenString == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
				c.Abort()
				return
			}

			token, err := adminService.VerifyIDToken(tokenString)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
				c.Abort()
				return
			}
			uid = token.UID
			c.Set("uid", uid)
		}

		// 2. Check Admin DB
		admin, err := adminService.GetProfileByUID(uid.(string))
		if err != nil || admin == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Not an admin"})
			c.Abort()
			return
		}

		fmt.Printf("AdminMiddleware: Found Admin - ID: %d (Type: %T), UID: %s\n", admin.ID, admin.ID, admin.UID)

		// Set context
		c.Set("admin_id", admin.ID)
		//c.Set("admin_id", int(admin.ID))

		c.Set("role", admin.Role)
		c.Set("email", admin.Email)
		c.Set("department", admin.Department)
		c.Next()
	}
}

// CustomerMiddleware checks if the authenticated user exists in the Customer table.
func CustomerMiddleware(adminService service.AdminService, customerService service.CustomerService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Verify Token (or get from context if chained)
		uid, exists := c.Get("uid")
		if !exists {
			// If not in context, verify header manually
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
				c.Abort()
				return
			}
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
				c.Abort()
				return
			}
			token, err := adminService.VerifyIDToken(parts[1])
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
				c.Abort()
				return
			}
			uid = token.UID
			c.Set("uid", uid)
		}

		// 2. Check Customer DB
		customer, err := customerService.GetCustomerByUID(uid.(string))
		if err != nil || customer == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Not a customer"})
			c.Abort()
			return
		}

		// Set context
		c.Set("customer_id", customer.ID)
		c.Set("email", customer.Email)
		c.Set("role", "customer")
		c.Next()
	}
}

// SuperAdminMiddleware checks if the user has 'super_admin' role.
// Must be used AFTER AdminMiddleware.
func SuperAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Unknown role"})
			c.Abort()
			return
		}

		if role != "super_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Super Admin only"})
			c.Abort()
			return
		}

		c.Next()
	}
}
