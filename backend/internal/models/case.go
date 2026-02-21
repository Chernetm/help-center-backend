package models

type Case struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"not null" json:"name"`
	Description string `gorm:"not null" json:"description"`
	Department  string `gorm:"not null" json:"department"`
}
