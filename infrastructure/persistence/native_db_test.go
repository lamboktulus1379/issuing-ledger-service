package persistence

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewNativeDb(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error %v", err)
	}

	db.Close()
	tests := []struct {
		name    string
		want    *sql.DB
		wantErr bool
	}{
		{
			name:    "Test #1",
			want:    db,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewNativeDb()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNativeDb() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// TODO: Fix this test
			// if !reflect.DeepEqual(got, tt.want) {
			// 	t.Errorf("NewNativeDb() = %v, want %v", got, tt.want)
			// }
		})
	}
}
