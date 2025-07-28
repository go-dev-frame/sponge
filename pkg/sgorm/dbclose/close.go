// Package dbclose provides a function to close gorm db.
package dbclose

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gorm"
)

// Close close gorm db
func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	checkInUse(sqlDB, time.Second*5)

	return sqlDB.Close()
}

func checkInUse(sqlDB *sql.DB, duration time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	for {
		select {
		case <-time.After(time.Millisecond * 250):
			if v := sqlDB.Stats().InUse; v == 0 {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
