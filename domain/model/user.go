package model

import (
	"time"

	"github.com/golang-jwt/jwt"
)

type User struct {
	ID        int64     `gorm:"primaryKey;column:id;type:bigint(20);not null"                                     json:"id"`
	Name      string    `gorm:"column:name;type:varchar(45)"`
	UserName  string    `gorm:"column:user_name;type:varchar(45)"`
	Password  string    `gorm:"column:password;type:varchar(225)"`
	Email     string    `                                                                                         json:"email"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	CreatedBy int64     `gorm:"column:created_by;type:varchar(225);not null"                                      json:"created_by,omitempty"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedBy int64     `gorm:"column:updated_by;type:int;not null"                                               json:"updated_at,omitempty"`
}

type UserClaims struct {
	UserName string `json:"user_name"`
	jwt.StandardClaims
}
