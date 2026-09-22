package persistence

import (
	"context"
	"database/sql"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/repository"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
)

const (
	ErrorPreparingStatement = "Error while prepare statement"
	ErrorClosingStatement   = "Error while close statement"
)

type UserRepository struct {
	sqlDB *sql.DB
}

func NewUserRepository(sqlDB *sql.DB) repository.IUser {
	return &UserRepository{sqlDB}
}

func (userRepository *UserRepository) GetById(ctx context.Context, id int) (model.User, error) {
	var user model.User

	statement, err := userRepository.sqlDB.PrepareContext(
		ctx,
		`SELECT u.id, u.name, u.user_name, u.password, u.created_at, u.updated_at 
	FROM user AS u 
	WHERE u.id = ?`,
	)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error(ErrorPreparingStatement)
		return user, err
	}
	defer func(statement *sql.Stmt) {
		err := statement.Close()
		if err != nil {
			logger.GetLogger().WithField("error", err).Error(ErrorClosingStatement)
		}
	}(statement)

	result := statement.QueryRow(id)
	err = result.Scan(
		&user.ID,
		&user.Name,
		&user.UserName,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while query")
		return user, err
	}

	return user, nil
}

func (userRepository *UserRepository) GetByUserName(
	ctx context.Context,
	userName string,
) (model.User, error) {
	var user model.User

	statement, err := userRepository.sqlDB.PrepareContext(
		ctx,
		`SELECT u.id, u.name, u.user_name, u.password, u.created_at, u.updated_at 
	FROM user AS u 
	WHERE u.user_name = ?`,
	)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error(ErrorPreparingStatement)
		return user, err
	}
	defer func(statement *sql.Stmt) {
		err := statement.Close()
		if err != nil {
			logger.GetLogger().WithField("error", err).Error(ErrorClosingStatement)
		}
	}(statement)

	result := statement.QueryRow(userName)
	err = result.Scan(
		&user.ID,
		&user.Name,
		&user.UserName,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while query")
		return user, err
	}

	return user, nil
}

func (userRepository *UserRepository) CreateUser(ctx context.Context, user model.User) error {
	statement, err := userRepository.sqlDB.PrepareContext(
		ctx,
		`INSERT INTO user (name, user_name, password) VALUES (?, ?, ?)`,
	)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error(ErrorPreparingStatement)
		return err
	}
	defer func(statement *sql.Stmt) {
		err := statement.Close()
		if err != nil {
			logger.GetLogger().WithField("error", err).Error(ErrorClosingStatement)
		}
	}(statement)

	_, err = statement.Exec(user.Name, user.UserName, user.Password)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error execute query")
		return err
	}

	return nil
}
