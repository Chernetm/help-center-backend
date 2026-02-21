package models

type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Role        string `json:"role"`
	PhoneNumber string `json:"phoneNumber"`
	Department  string `json:"department"`
	Address     string `json:"address"`
	Image       string `json:"image"`
}
type ProfileUpdateRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Role        string `json:"role"`
	PhoneNumber string `json:"phoneNumber"`
	Department  string `json:"department"`
	Address     string `json:"address"`
	Image       string `json:"image"`
}
type UserProfileLocal struct {
	ID          uint64 `json:"id"`
	UID         string
	Email       string
	FirstName   string
	LastName    string
	Role        string
	PhoneNumber string
	Department  string
	Address     string
	Image       string
}

type UserProfileResponse struct {
	ID          uint64 `json:"id"`
	UID         string `json:"uid"`
	Email       string `json:"email"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Role        string `json:"role"`
	PhoneNumber string `json:"phoneNumber"`
	Address     string `json:"address"`
	Department  string `json:"department"`
	Image       string `json:"image"`
}
