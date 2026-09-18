package repository_test

import (
	"testing"

	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/persistence"
)

func Test_GetById(t *testing.T) {
	db, _ := persistence.NewNativeDb()
	err := db.Close()
	if err != nil {
		return
	}
}
