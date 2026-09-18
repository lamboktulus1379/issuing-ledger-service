package model

import (
	"time"
)

type Project struct {
	Id          int       `json:"id"          bson:"id"`
	Name        string    `                   bson:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `                   bson:"createdAt" gorm:"autoCreateTime;index"`
	UpdatedAt   time.Time `                   bson:"updatedAt" gorm:"autoUpdateTime;index"`
}
