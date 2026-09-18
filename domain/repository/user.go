package repository

import (
	"context"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
)

type IUser interface {
	GetById(ctx context.Context, id int) (model.User, error)
	GetByUserName(ctx context.Context, userName string) (model.User, error)
	CreateUser(ctx context.Context, user model.User) error
}
