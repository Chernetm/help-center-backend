package models

type Customer struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	UID         string `gorm:"size:191;uniqueIndex" json:"uid"`
	Name        string `gorm:"size:191;not null" json:"name"`
	Password    string `gorm:"not null" json:"-"` // Do NOT expose in JSON
	PhoneNumber string `gorm:"size:191;unique;not null" json:"phoneNumber"`
	Email       string `gorm:"size:191;unique" json:"email"`
	Status      string `gorm:"size:20;default:'active'" json:"status"`          // active, inactive, blocked
	Role        string `gorm:"size:20;default:'customer';not null" json:"role"` // customer, admin?? Wait user said "role status change separetely". Assuming Customer has a Role too or user implies generic Role. But models.Customer is likely just customer type. Wait. Models.Admin exists separately.
}
type CustomerRegisterRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=191"`
	Email       string `json:"email" binding:"required,email"`
	PhoneNumber string `json:"phoneNumber" binding:"required"`
	Password    string `json:"password" binding:"required,min=6"`
}
