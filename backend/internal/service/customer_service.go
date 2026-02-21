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

	"firebase.google.com/go/v4/auth"
)

type CustomerService interface {
	RegisterCustomer(req *models.CustomerRegisterRequest) (*models.Customer, error)
	Login(email, password string) (*models.Customer, string, error)
	GetCustomerByEmail(email string) (*models.Customer, error)
	GetCustomerByUID(uid string) (*models.Customer, error)
	GetCustomerByID(id uint) (*models.Customer, error)
	GetAllCustomers() ([]models.Customer, error)
	UpdateCustomer(customer *models.Customer) error
	UpdateCustomerStatus(id uint, status string) error
	UpdateCustomerRole(id uint, role string) error
	DeleteCustomer(id uint) error
	SyncFirebaseUser(uid, email, name string) (*models.Customer, error)
	ForgotPassword(email string) error
	ChangePassword(uid string, newPassword string) error
}

type customerService struct {
	repo     repository.CustomerRepository
	firebase *FirebaseService
	cfg      *config.Config
}

func NewCustomerService(repo repository.CustomerRepository, fs *FirebaseService, cfg *config.Config) CustomerService {
	return &customerService{
		repo:     repo,
		firebase: fs,
		cfg:      cfg,
	}
}

func (s *customerService) RegisterCustomer(req *models.CustomerRegisterRequest) (*models.Customer, error) {
	ctx := context.Background()

	// 1. Create user in Firebase Auth
	params := (&auth.UserToCreate{}).
		Email(req.Email).
		Password(req.Password).
		DisplayName(req.Name)

	userRecord, err := s.firebase.AuthClient.CreateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create user in firebase: %v", err)
	}

	// 2. Create local customer profile
	customer := &models.Customer{
		UID:         userRecord.UID,
		Email:       req.Email,
		Name:        req.Name,
		PhoneNumber: req.PhoneNumber,
		Status:      "active",
		Role:        "customer",
	}

	// Note: Customer model doesn't have UID or FirstName/LastName separate fields yet based on previous view.
	// Let's check Customer model again. It has Name, Email, PhoneNumber, Password, Status, Role.
	// Wait, previous Customer struct:
	// ID uint, Name string, Password string, PhoneNumber string, Email string, Status string, Role string.
	// It does NOT have UID. This might be an issue if we want to link with Firebase UID.
	// Admin model had UID. Customer model relies on ID (uint).
	// But successful login returns IDToken from Firebase.
	// If we use Firebase for Auth, we should probably store UID or at least link via Email.
	// Admin service links by UID.
	// For now I will link by Email as Customer model doesn't have UID shown in previous steps.
	// Or I should add UID to Customer model?
	// The user request is "make ALL methods ... like admin_service".
	// Admin service uses UID.
	// If I modify Customer model to have UID, it would be better.
	// But let's stick to existing fields first. Email is unique.
	// Or I should add UID to Customer model?
	// The user request is "make ALL methods ... like admin_service".
	// Admin service uses UID.
	// If I modify Customer model to have UID, it would be better.
	// But let's stick to existing fields first. Email is unique.

	if err := s.repo.Create(customer); err != nil {
		return nil, errors.New("failed to save customer profile")
	}

	return customer, nil
}

func (s *customerService) Login(email, password string) (*models.Customer, string, error) {
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
		return nil, "", errors.New("firebase auth failed")
	}

	var result struct {
		IDToken string `json:"idToken"`
		Email   string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", err
	}

	customer, err := s.repo.FindByEmail(result.Email)
	if err != nil {
		return nil, "", errors.New("customer profile not found locally")
	}

	return customer, result.IDToken, nil
}

func (s *customerService) GetCustomerByEmail(email string) (*models.Customer, error) {
	return s.repo.FindByEmail(email)
}

func (s *customerService) GetCustomerByUID(uid string) (*models.Customer, error) {
	return s.repo.FindByUID(uid)
}

func (s *customerService) GetCustomerByID(id uint) (*models.Customer, error) {
	return s.repo.FindByID(id)
}

func (s *customerService) GetAllCustomers() ([]models.Customer, error) {
	return s.repo.FindAll()
}

func (s *customerService) UpdateCustomer(customer *models.Customer) error {
	return s.repo.Update(customer)
}

func (s *customerService) UpdateCustomerStatus(id uint, status string) error {
	return s.repo.UpdateStatus(id, status)
}

func (s *customerService) UpdateCustomerRole(id uint, role string) error {
	return s.repo.UpdateRole(id, role)
}

func (s *customerService) DeleteCustomer(id uint) error {

	return s.repo.Delete(id)
}

func (s *customerService) SyncFirebaseUser(uid, email, name string) (*models.Customer, error) {
	// Check if exists
	existing, err := s.repo.FindByUID(uid)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Create new
	customer := &models.Customer{
		UID:    uid,
		Email:  email,
		Name:   name,
		Status: "active",
		Role:   "customer",
	}

	if err := s.repo.Create(customer); err != nil {
		return nil, err
	}

	return customer, nil
}

func (s *customerService) ForgotPassword(email string) error {
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

func (s *customerService) ChangePassword(uid string, newPassword string) error {
	ctx := context.Background()
	params := (&auth.UserToUpdate{}).
		Password(newPassword)

	_, err := s.firebase.AuthClient.UpdateUser(ctx, uid, params)
	if err != nil {
		return fmt.Errorf("failed to update password in firebase: %v", err)
	}

	return nil
}
