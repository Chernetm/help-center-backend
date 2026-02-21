package service

import (
	"bytes"
	"context"
	"customer-help-center-backend/internal/config"
	"customer-help-center-backend/internal/models"
	"customer-help-center-backend/internal/repository"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"firebase.google.com/go/v4/auth"
)

type AdminService interface {
	RegisterAdmin(req *models.RegisterRequest) (*models.UserProfileResponse, error)
	Login(email, password string) (*models.UserProfileLocal, string, error)
	GetProfileByUID(uid string) (*models.UserProfileResponse, error)
	UpdateProfile(uid string, req *models.RegisterRequest) (*models.UserProfileResponse, error)
	VerifyIDToken(token string) (*auth.Token, error)
	GetAdminByID(id uint64) (*models.Admin, error)
	GetAgentTickets(agentID uint64) ([]models.Ticket, error)
	GetAllUsers() ([]models.Admin, error)
	DeleteAdmin(uid string) error
	UpdateAdmin(uid string, req *models.Admin) (*models.Admin, error)
	StartWorkSession(adminID uint64) (*models.WorkSession, error)
	EndWorkSession(adminID uint64) error
	EndWorkSessionByID(sessionID uint) error
	GetAgentPerformance() ([]models.AgentPerformance, error)
	SetOnlineStatus(adminID uint64, isOnline bool) error
	ForgotPassword(email string) error
	ChangePassword(uid string, newPassword string) error
}

type adminService struct {
	repo            repository.AdminRepository
	workSessionRepo repository.WorkSessionRepository
	firebase        *FirebaseService
	cfg             *config.Config
}

func NewAdminService(
	repo repository.AdminRepository,
	workSessionRepo repository.WorkSessionRepository,
	fs *FirebaseService,
	cfg *config.Config,
) AdminService {
	return &adminService{
		repo:            repo,
		workSessionRepo: workSessionRepo,
		firebase:        fs,
		cfg:             cfg,
	}
}

func (s *adminService) RegisterAdmin(req *models.RegisterRequest) (*models.UserProfileResponse, error) {
	ctx := context.Background()

	// 1. Create user in Firebase Auth
	params := (&auth.UserToCreate{}).
		Email(req.Email).
		Password(req.Password).
		DisplayName(fmt.Sprintf("%s %s", req.FirstName, req.LastName))

	userRecord, err := s.firebase.AuthClient.CreateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create user in firebase: %v", err)
	}

	// 2. Create local admin profile
	admin := &models.Admin{
		UID:         userRecord.UID,
		Email:       req.Email,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		PhoneNumber: req.PhoneNumber,
		Address:     req.Address,
		Image:       req.Image,
		Role:        "super-admin", // Default role for first admin; in real app, consider role assignment logic
		Department:  "",
		IsOnline:    false,
		// Password is not stored locally
	}

	// Ensure username is set if required by DB constraint

	if err := s.repo.Create(admin); err != nil {
		// Ideally delete firebase user here to rollback
		fmt.Printf("FATAL: Failed to create local admin profile for UID %s: %v\n", userRecord.UID, err)
		return nil, errors.New("failed to save admin profile")
	}

	// 3. Return response
	response := &models.UserProfileResponse{
		UID:         userRecord.UID,
		Email:       admin.Email,
		FirstName:   admin.FirstName,
		LastName:    admin.LastName,
		PhoneNumber: admin.PhoneNumber,
		Address:     admin.Address,
	}

	return response, nil
}

func (s *adminService) Login(email, password string) (*models.UserProfileLocal, string, error) {
	url := "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=" + s.cfg.FirebaseAPIKey

	payload := map[string]interface{}{
		"email":             email,
		"password":          password,
		"returnSecureToken": true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)

		return nil, "", fmt.Errorf(
			"firebase auth failed (%d): %s",
			resp.StatusCode,
			errResp.Error.Message,
		)
	}

	// Parse Firebase response
	var result struct {
		IDToken string `json:"idToken"`
		LocalID string `json:"localId"`
		Email   string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", err
	}

	// Lookup user in DB
	admin, err := s.repo.FindByEmail(result.Email)
	if err != nil {
		return nil, "", errors.New("admin profile not found locally")
	}

	userProfile := &models.UserProfileLocal{
		ID:          admin.ID,
		UID:         admin.UID,
		Email:       admin.Email,
		FirstName:   admin.FirstName,
		LastName:    admin.LastName,
		Role:        admin.Role,
		PhoneNumber: admin.PhoneNumber,
		Address:     admin.Address,
	}

	return userProfile, result.IDToken, nil
}

func (s *adminService) GetProfileByUID(uid string) (*models.UserProfileResponse, error) {
	admin, err := s.repo.FindByUID(uid)
	if err != nil {
		return nil, errors.New("admin profile not found")
	}

	response := &models.UserProfileResponse{
		ID:          admin.ID,
		UID:         admin.UID,
		Email:       admin.Email,
		FirstName:   admin.FirstName,
		LastName:    admin.LastName,
		Role:        admin.Role,
		PhoneNumber: admin.PhoneNumber,
		Address:     admin.Address,
	}
	return response, nil
}

func (s *adminService) UpdateProfile(uid string, req *models.RegisterRequest) (*models.UserProfileResponse, error) {
	admin, err := s.repo.FindByUID(uid)
	if err != nil {
		return nil, errors.New("admin profile not found")
	}

	if req.FirstName != "" {
		admin.FirstName = req.FirstName
	}
	if req.LastName != "" {
		admin.LastName = req.LastName
	}
	if req.PhoneNumber != "" {
		admin.PhoneNumber = req.PhoneNumber
	}
	if req.Address != "" {
		admin.Address = req.Address
	}
	if req.Department != "" {
		admin.Department = req.Department
	}

	if err := s.repo.Update(admin); err != nil {
		return nil, errors.New("failed to update admin profile")
	}

	// Update Firebase DisplayName
	ctx := context.Background()
	params := (&auth.UserToUpdate{}).
		DisplayName(fmt.Sprintf("%s %s", admin.FirstName, admin.LastName))

	if _, err := s.firebase.AuthClient.UpdateUser(ctx, uid, params); err != nil {
		fmt.Printf("Warning: Failed to update Firebase DisplayName for UID %s: %v\n", uid, err)
	}

	response := &models.UserProfileResponse{
		ID:          admin.ID,
		Email:       admin.Email,
		FirstName:   admin.FirstName,
		LastName:    admin.LastName,
		Role:        admin.Role,
		PhoneNumber: admin.PhoneNumber,
		Department:  admin.Department,
		Address:     admin.Address,
	}
	return response, nil
}

//	func (s *adminService) VerifyIDToken(token string) (*auth.Token, error) {
//		ctx := context.Background()
//		return s.firebase.AuthClient.VerifyIDToken(ctx, token)
//	}
func (s *adminService) VerifyIDToken(token string) (*auth.Token, error) {
	ctx := context.Background()
	return s.firebase.AuthClient.VerifyIDToken(ctx, token)
}

func (s *adminService) GetAdminByID(id uint64) (*models.Admin, error) {
	return s.repo.FindByID(id)
}

func (s *adminService) GetAgentTickets(adminID uint64) ([]models.Ticket, error) {
	return nil, errors.New("not implemented")
}

func (s *adminService) GetAllUsers() ([]models.Admin, error) {
	return s.repo.FindAll()
}

func (s *adminService) DeleteAdmin(uid string) error {
	// 1. Delete from local DB
	if err := s.repo.Delete(uid); err != nil {
		return err
	}

	// 2. Delete from Firebase
	ctx := context.Background()
	if err := s.firebase.AuthClient.DeleteUser(ctx, uid); err != nil {
		// Log error but don't fail the whole operation if local DB is already cleaned?
		// Or maybe we should return error? Ideally distributed transaction but KISS.
		fmt.Printf("Warning: Failed to delete Firebase user %s: %v\n", uid, err)
		// For now, let's return the error so the caller knows something went wrong.
		// Although inconsistencies might occur.
		return err
	}

	return nil
}

func (s *adminService) UpdateAdmin(uid string, req *models.Admin) (*models.Admin, error) {
	admin, err := s.repo.FindByUID(uid)
	if err != nil {
		return nil, errors.New("admin profile not found")
	}

	// Update fields if they are provided (not empty)
	if req.Role != "" {
		admin.Role = req.Role
	}
	if req.Status != "" {
		admin.Status = req.Status
	}
	if req.Department != "" {
		admin.Department = req.Department
	}
	// Add other fields as necessary, e.g., Name, Phone? Plan mentioned Role, Status, Department.

	if err := s.repo.Update(admin); err != nil {
		return nil, errors.New("failed to update admin profile")
	}

	return admin, nil
}

func (s *adminService) StartWorkSession(adminID uint64) (*models.WorkSession, error) {
	session := &models.WorkSession{
		AdminID: adminID,
	}
	err := s.workSessionRepo.Create(session)
	return session, err
}

func (s *adminService) EndWorkSession(adminID uint64) error {
	return s.workSessionRepo.EndActiveSession(adminID)
}

func (s *adminService) EndWorkSessionByID(sessionID uint) error {
	return s.workSessionRepo.EndSessionByID(sessionID)
}

func (s *adminService) GetAgentPerformance() ([]models.AgentPerformance, error) {
	admins, err := s.repo.FindAgentsFullDetails()
	if err != nil {
		return nil, err
	}

	performance := []models.AgentPerformance{}
	now := time.Now()

	// Helper to get start of period
	startOfWeek := now.AddDate(0, 0, -int(now.Weekday())) // active week starting sunday/monday? assuming sunday as 0
	if now.Weekday() == time.Monday {
		startOfWeek = now.Truncate(24*time.Hour).AddDate(0, 0, -6) // Example: previous 6 days + today
	} else {
		// Just last 7 days? Or calendar week? Commonly "this week"
		// Let's use simple "Last 7 days" or "Current Week"
		// Using "Current Week" (from last Sunday)
		startOfWeek = now.AddDate(0, 0, -int(now.Weekday()))
	}
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())

	for _, admin := range admins {
		totalTickets := len(admin.Tickets)
		closedTickets := 0
		uniqueCustomers := make(map[uint]bool)
		var totalRating float64
		var ratingCount int

		for _, t := range admin.Tickets {
			if t.Status == "closed" {
				closedTickets++
			}
			uniqueCustomers[t.CustomerID] = true
			for _, r := range t.Ratings {
				totalRating += float64(r.Score)
				ratingCount++
			}
		}

		var avgRating *float64
		if ratingCount > 0 {
			avg := totalRating / float64(ratingCount)
			avgVal := float64(int(avg*10)) / 10 // Round to 1 decimal
			avgRating = &avgVal
		}

		// Calculate Durations
		var weekly, monthly, annually int
		for _, sess := range admin.WorkSessions {
			if sess.Duration == nil {
				continue
			}
			dur := *sess.Duration // minutes

			// Check range
			if sess.StartedAt.After(startOfWeek) {
				weekly += dur
			}
			if sess.StartedAt.After(startOfMonth) {
				monthly += dur
			}
			if sess.StartedAt.After(startOfYear) {
				annually += dur
			}
		}

		performance = append(performance, models.AgentPerformance{
			ID:               admin.ID,
			Name:             fmt.Sprintf("%s %s", admin.FirstName, admin.LastName),
			Department:       admin.Department,
			IsOnline:         admin.IsOnline,
			TotalTickets:     totalTickets,
			ClosedTickets:    closedTickets,
			PendingTickets:   totalTickets - closedTickets,
			AvgRating:        avgRating,
			CustomersServed:  len(uniqueCustomers),
			DurationWeekly:   weekly,
			DurationMonthly:  monthly,
			DurationAnnually: annually,
		})
	}

	return performance, nil
}

func (s *adminService) SetOnlineStatus(adminID uint64, isOnline bool) error {
	return s.repo.SetOnlineStatus(adminID, isOnline)
}

func (s *adminService) ForgotPassword(email string) error {
	url := "https://identitytoolkit.googleapis.com/v1/accounts:sendOobCode?key=" + s.cfg.FirebaseAPIKey

	payload := map[string]interface{}{
		"email":       email,
		"requestType": "PASSWORD_RESET",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("firebase error: %s", errResp.Error.Message)
	}

	return nil
}

func (s *adminService) ChangePassword(uid string, newPassword string) error {
	ctx := context.Background()
	params := (&auth.UserToUpdate{}).
		Password(newPassword)

	_, err := s.firebase.AuthClient.UpdateUser(ctx, uid, params)
	if err != nil {
		return fmt.Errorf("failed to update password in firebase: %v", err)
	}

	return nil
}
