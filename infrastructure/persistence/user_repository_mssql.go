package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/repository"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
)

// UserRepositoryMSSQL is a SQL Server implementation of IUser using database/sql.
type UserRepositoryMSSQL struct{ db *sql.DB }

func NewUserRepositoryMSSQL(db *sql.DB) repository.IUser { return &UserRepositoryMSSQL{db} }

func (r *UserRepositoryMSSQL) GetById(ctx context.Context, id int) (model.User, error) {
	var u model.User
	row := r.db.QueryRowContext(ctx, `SELECT id, name, user_name, password, created_at, updated_at FROM dbo.[users] WHERE id = @p1`, id)
	if err := row.Scan(&u.ID, &u.Name, &u.UserName, &u.Password, &u.CreatedAt, &u.UpdatedAt); err != nil {
		logger.GetLogger().WithField("error", err).Error("mssql: query user by id failed")
		return u, err
	}
	return u, nil
}

func (r *UserRepositoryMSSQL) GetByUserName(ctx context.Context, userName string) (model.User, error) {
	var u model.User
	row := r.db.QueryRowContext(ctx, `SELECT id, name, user_name, password, created_at, updated_at FROM dbo.[users] WHERE user_name = @p1`, userName)
	if err := row.Scan(&u.ID, &u.Name, &u.UserName, &u.Password, &u.CreatedAt, &u.UpdatedAt); err != nil {
		logger.GetLogger().WithField("error", err).Error("mssql: query user by username failed")
		return u, err
	}
	return u, nil
}

func (r *UserRepositoryMSSQL) CreateUser(ctx context.Context, user model.User) error {
	createdAt := user.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO dbo.[users] (name, user_name, password, created_at, updated_at) VALUES (@p1, @p2, @p3, @p4, SYSDATETIME())`, user.Name, user.UserName, user.Password, createdAt)
	if err != nil {
		logger.GetLogger().WithFields(map[string]interface{}{
			"error":     err,
			"user_name": user.UserName,
		}).Error("mssql: create user failed")
	}
	return err
}
