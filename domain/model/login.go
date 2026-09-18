package model

type ReqLogin struct {
	UserName string `json:"user_name" binding:"required"`
	Password string `json:"password"  binding:"required"`
}

type ReqRegister struct {
	Name     string `json:"name"      binding:"required"`
	UserName string `json:"user_name" binding:"required"`
	Password string `json:"password"  binding:"required"`
}
