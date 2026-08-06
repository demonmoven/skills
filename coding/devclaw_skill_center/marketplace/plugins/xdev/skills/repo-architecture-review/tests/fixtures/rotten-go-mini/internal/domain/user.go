package domain

import (
	"example.com/rotten/internal/infra"
)

func GetUser(userID int) string {
	db := infra.ReadDB()
	return db.Query("SELECT * FROM users WHERE id = ?", userID)
}
