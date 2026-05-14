package model

import "time"

// User maps to the "user" table.
type User struct {
	ID            string     `json:"id" gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	Username      string     `json:"username" gorm:"column:username"`
	Email         string     `json:"email" gorm:"column:email"`
	Phone         *string    `json:"phone,omitempty" gorm:"column:phone"`
	PasswordHash  string     `json:"-" gorm:"column:password_hash"`
	EmailVerified bool       `json:"email_verified" gorm:"column:email_verified;default:false"`
	Role          string     `json:"role" gorm:"column:role;default:user"`
	Status        string     `json:"status" gorm:"column:status;default:active"`
	CreatedAt     time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"column:updated_at"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty" gorm:"column:last_login_at"`
}

func (User) TableName() string {
	return "user"
}
