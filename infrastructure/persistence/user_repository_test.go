package persistence

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
)

// TestUserRepository_GetById_Fixed tests the GetById method with isolated mock
func TestUserRepository_GetById_Fixed(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repository := NewUserRepository(db)

	loc, _ := time.LoadLocation("Asia/Jakarta")
	createdAtTime, _ := time.Parse(
		"2006-01-02 15:04:05.999999999+07 MST",
		"2023-09-04 01:02:10.911651+07 WIB",
	)
	updatedAtTime, _ := time.Parse(
		"2006-01-02 15:04:05.999999999+07 MST",
		"2023-09-04 01:02:10.911651+07 WIB",
	)

	var (
		ID        = 1
		Name      = "Lambok Tulus Simamora"
		UserName  = "lamboktulus1379"
		Password  = "a252f77af72638ea5a0f9e5fbe5f2b2e"
		CreatedAt = createdAtTime.In(loc)
		UpdatedAt = updatedAtTime.In(loc)
	)

	mock.ExpectPrepare(regexp.QuoteMeta(`SELECT u.id, u.name, u.user_name, u.password, u.created_at, u.updated_at 
	FROM public.user AS u 
	WHERE u.id = $1`)).
		ExpectQuery().WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "user_name", "password", "created_at", "updated_at"}).
			AddRow(ID, Name, UserName, Password, CreatedAt, UpdatedAt))

	res, err := repository.GetById(context.Background(), 1)
	expected := model.User{
		ID:        1,
		Name:      "Lambok Tulus Simamora",
		UserName:  "lamboktulus1379",
		Password:  "a252f77af72638ea5a0f9e5fbe5f2b2e",
		CreatedAt: CreatedAt,
		UpdatedAt: UpdatedAt,
	}

	require.NoError(t, err)
	require.Equal(t, expected, res)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUserRepository_GetByUserName_Fixed tests the GetByUserName method with isolated mock
func TestUserRepository_GetByUserName_Fixed(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repository := NewUserRepository(db)

	loc, _ := time.LoadLocation("Asia/Jakarta")
	createdAtTime, _ := time.Parse(
		"2006-01-02 15:04:05.999999999+07 MST",
		"2023-09-04 01:02:10.911651+07 WIB",
	)
	updatedAtTime, _ := time.Parse(
		"2006-01-02 15:04:05.999999999+07 MST",
		"2023-09-04 01:02:10.911651+07 WIB",
	)

	var (
		ID        = 1
		Name      = "Lambok Tulus Simamora"
		UserName  = "lamboktulus1379"
		Password  = "a252f77af72638ea5a0f9e5fbe5f2b2e"
		CreatedAt = createdAtTime.In(loc)
		UpdatedAt = updatedAtTime.In(loc)
	)

	mock.ExpectPrepare(regexp.QuoteMeta(`SELECT u.id, u.name, u.user_name, u.password, u.created_at, u.updated_at 
	FROM public.user AS u 
	WHERE u.user_name = $1`)).
		ExpectQuery().WithArgs("lamboktulus1379").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "user_name", "password", "created_at", "updated_at"}).
			AddRow(ID, Name, UserName, Password, CreatedAt, UpdatedAt))

	res, err := repository.GetByUserName(context.Background(), "lamboktulus1379")
	expected := model.User{
		ID:        1,
		Name:      "Lambok Tulus Simamora",
		UserName:  "lamboktulus1379",
		Password:  "a252f77af72638ea5a0f9e5fbe5f2b2e",
		CreatedAt: CreatedAt,
		UpdatedAt: UpdatedAt,
	}

	require.NoError(t, err)
	require.Equal(t, expected, res)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUserRepository_CreateUser_Fixed tests the CreateUser method with isolated mock
func TestUserRepository_CreateUser_Fixed(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repository := NewUserRepository(db)

	var (
		Name     = "Lambok Tulus Simamora"
		UserName = "lamboktulus1379"
		Password = "a252f77af72638ea5a0f9e5fbe5f2b2e"
	)

	mock.ExpectPrepare(regexp.QuoteMeta(`INSERT INTO public.user (name, user_name, password) VALUES ($1, $2, $3)`)).
		ExpectExec().WithArgs(Name, UserName, Password).
		WillReturnResult(sqlmock.NewResult(1, 1))

	user := model.User{
		Name:     Name,
		UserName: UserName,
		Password: Password,
	}

	err = repository.CreateUser(context.Background(), user)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUserRepository_GetById_PrepareError tests error handling in GetById
func TestUserRepository_GetById_PrepareError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repository := NewUserRepository(db)

	mock.ExpectPrepare(regexp.QuoteMeta(`SELECT u.id, u.name, u.user_name, u.password, u.created_at, u.updated_at 
	FROM public.user AS u 
	WHERE u.id = $1`)).
		WillReturnError(fmt.Errorf("prepare error"))

	res, err := repository.GetById(context.Background(), 1)
	expected := model.User{}

	require.Error(t, err)
	require.Equal(t, expected, res)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUserRepository_GetByUserName_PrepareError tests error handling in GetByUserName
func TestUserRepository_GetByUserName_PrepareError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repository := NewUserRepository(db)

	mock.ExpectPrepare(regexp.QuoteMeta(`SELECT u.id, u.name, u.user_name, u.password, u.created_at, u.updated_at 
	FROM public.user AS u 
	WHERE u.user_name = $1`)).
		WillReturnError(fmt.Errorf("prepare error"))

	res, err := repository.GetByUserName(context.Background(), "lamboktulus1379")
	expected := model.User{}

	require.Error(t, err)
	require.Equal(t, expected, res)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUserRepository_CreateUser_PrepareError tests error handling in CreateUser
func TestUserRepository_CreateUser_PrepareError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repository := NewUserRepository(db)

	mock.ExpectPrepare(regexp.QuoteMeta(`INSERT INTO public.user (name, user_name, password) VALUES ($1, $2, $3)`)).
		WillReturnError(fmt.Errorf("prepare error"))

	user := model.User{
		Name:     "Test User",
		UserName: "testuser",
		Password: "testpass",
	}

	err = repository.CreateUser(context.Background(), user)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
