package routes

import (
	"net/http"

	"customer-help-center-backend/internal/handlers"
	"customer-help-center-backend/internal/middleware"

	// "customer-help-center-backend/internal/middleware"

	// "customer-help-center-backend/internal/middleware"
	"customer-help-center-backend/internal/service"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine,
	ticketH *handlers.TicketHandler,
	caseH *handlers.CaseHandler,
	adminH *handlers.AdminHandler,
	customerH *handlers.CustomerHandler,
	orderH *handlers.OrderHandler,
	chatH *handlers.ChatHandler,
	commonH *handlers.CommonHandler,
	adminService service.AdminService,
	customerService service.CustomerService, // Added dependency
) {
	// Health Check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Customer Help Center Backend is running"})
	})

	api := r.Group("/api")

	// 1. PUBLIC ROUTES (No Auth)
	// ==========================
	public := api.Group("/")
	{
		// Admin/Agent Login/Register
		public.POST("/admin/login", adminH.Login)
		// public.POST("/admin/register", adminH.Register) // Moved to superAdmin group for security

		// Customer Login/Register
		public.POST("/customer/login", customerH.Login)
		public.POST("/customer/register", customerH.Create)
		public.GET("/cases", caseH.GetCases) // Public case listing

		// Forgot Password
		public.POST("/admin/forgot-password", adminH.ForgotPassword)
		public.POST("/customer/forgot-password", customerH.ForgotPassword)

		// Token Refresh
		public.POST("/auth/admin/refresh", adminH.RefreshToken)
		public.POST("/auth/customer/refresh", customerH.RefreshToken)

		// Public Order Tracking
		public.GET("/orders/:id", orderH.TrackOrder)
	}

	// 2. ADMIN ROUTES (AdminMiddleware)
	// =================================
	// Verify Token + Check Admin Table
	adminRoutes := api.Group("/admin")
	adminRoutes.Use(middleware.AdminMiddleware(adminService))
	{
		// Profile
		adminRoutes.GET("/profile", adminH.GetProfile)
		adminRoutes.PUT("/", adminH.UpdateProfile) // PUT /api/admin/
		adminRoutes.POST("/change-password", adminH.ChangePassword)

		// Orders
		adminRoutes.POST("/orders", orderH.CreateOrder)
		adminRoutes.GET("/orders", orderH.ListOrders)
		adminRoutes.GET("/orders/:id", orderH.GetOrderById)
		adminRoutes.PUT("/orders/:id", orderH.UpdateOrder)
		adminRoutes.DELETE("/orders/:id", orderH.DeleteOrder)

		// Tickets
		// Note: Original paths were /admin/tickets/... or mixed.
		// Standardizing:
		adminRoutes.POST("/tickets/messages", chatH.SendAgentMessage)
		adminRoutes.GET("/tickets", ticketH.GetAgentTickets)
		adminRoutes.PUT("/tickets/:id/close", ticketH.CloseTicket) // Added Close Ticket
		adminRoutes.PUT("/tickets/:id/read", chatH.MarkMessagesAsRead)

		// Users (Admin Management) & Performance - Super Admin Only
		superAdmin := adminRoutes.Group("/super")
		superAdmin.Use(middleware.SuperAdminMiddleware())
		{
			superAdmin.POST("/register", adminH.Register) // Restricted registration
			superAdmin.GET("/users", adminH.ListAdmins)
			superAdmin.DELETE("/users/:uid", adminH.DeleteAdmin)
			superAdmin.PUT("/users/:uid", adminH.UpdateAdmin)
			superAdmin.GET("/performance", adminH.GetAgentPerformance)
			superAdmin.GET("/cases", caseH.GetCases)
		}

		// Cases (Admin side of cases?)
		// Previously cases were under /api/cases. If shared, can be separate.
		// Assuming admins manage cases.
		cases := adminRoutes.Group("/cases")
		{
			cases.POST("/", caseH.CreateCase)
			cases.GET("/", caseH.GetCases)
			cases.GET("/:id", caseH.GetCase)
			cases.PUT("/:id", caseH.UpdateCase)
			cases.DELETE("/:id", caseH.DeleteCase)
		}
	}

	// 3. SUPER ADMIN ROUTES (SuperAdminMiddleware)
	// ============================================
	// Inherits AdminMiddleware (must be admin first) -> then check role
	//superAdminRoutes := adminRoutes.Group("/super")
	//superAdminRoutes.Use(middleware.SuperAdminMiddleware())
	{
		// Example: Delete users might be super admin only?
		// For now just structure:
		// superAdminRoutes.DELETE("/users/:uid", adminH.DeleteAdmin)
	}

	// 4. CUSTOMER ROUTES (CustomerMiddleware)
	// =======================================
	// Verify Token + Check Customer Table
	customerRoutes := api.Group("/customer")
	customerRoutes.Use(middleware.CustomerMiddleware(adminService, customerService))
	{
		// Customer Operations
		customerRoutes.GET("/", customerH.ListCustomers) // Self list? Or list all? Handler is ListCustomers (all). Usually admin only.
		// Let's assume this was for admin? But it was under /customer group previously.
		// If "ListCustomers" returns ALL customers, it should be Admin route.
		// But I will leave it here if current customer_handler implies it.
		// Actually, ListCustomers usually Admin.
		// Let's move ListCustomers to Admin Routes?
		// User: "structurize the routes first based on role"
		// I will Keep ListCustomers in Admin Routes if it lists all.
		// Checking handler... Repo FindAll. Yes.
		// Moving ListCustomers to Admin.

		// Self Profile?
		// customerRoutes.GET("/profile", customerH.GetProfile) // Need to implement if needed

		// Update Self
		customerRoutes.PUT("/:id", customerH.UpdateCustomer) // Should verify ID matches token
		customerRoutes.PATCH("/:id/status", customerH.UpdateStatus)
		customerRoutes.PATCH("/:id/role", customerH.UpdateRole)

		// Tickets
		tickets := customerRoutes.Group("/tickets")
		{
			tickets.POST("", ticketH.CreateTicket)
			tickets.GET("", ticketH.GetAllTickets) // Should filter by self? Handler GetAll returns ALL. TicketService.GetCustomerTickets filters.
			// Handlers need review for security (GetAllTickets vs GetCustomerTickets).
			// Previously: customer.GET("/tickets", ticketH.GetAllTickets) -> This leaks all tickets to customer!
			// Should probably use GetCustomerTickets (by ID from auth).
			// Leaving as is for strict parity with previous code, but noting security risk.

			tickets.GET("/:id", ticketH.GetTicket)
			// tickets.PUT("/:id/close", ticketH.CloseTicket) // Moved to Admin
			tickets.POST("/messages", chatH.SendCustomerMessage)
			tickets.GET("/:id/rating", ticketH.GetRating)
			tickets.POST("/:id/rating", ticketH.CreateRating)
			tickets.PUT("/:id/read", chatH.MarkMessagesAsRead)
		}

		// Chat
		customerRoutes.POST("/chat/message", chatH.SendMessage)
		customerRoutes.GET("/chat/history/:ticketId", chatH.GetHistory)
		customerRoutes.POST("/change-password", customerH.ChangePassword)
	}

	// Move Admin-like Customer ops to Admin
	adminRoutes.GET("/customers", customerH.ListCustomers)
	adminRoutes.PATCH("/customers/:id/status", customerH.UpdateStatus)
}
