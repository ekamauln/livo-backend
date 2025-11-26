package utilities

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// GenerateComplainCode generates a complain code with format: YYYYMMDD + first 2 chars of username + 3-digit auto increment
// Example: 20251008SA001, 20251008SA002, etc.
func GenerateComplainCode(db *gorm.DB, username string) string {
	// Get current date in YYYYMMDD format
	now := time.Now()
	datePrefix := now.Format("20060102")

	// Get first 2 characters of username (uppercase)
	var userPrefix string
	if len(username) >= 2 {
		userPrefix = strings.ToUpper(username[:2])
	} else if len(username) == 1 {
		userPrefix = strings.ToUpper(username + "X") // Pad with 'X' if only 1 char
	} else {
		userPrefix = "XX" // Default if username is empty
	}

	// Count complains for current date to get auto increment number
	var count int64
	startOfDay := now.Format("2006-01-02 00:00:00")
	endOfDay := now.Format("2006-01-02 23:59:59")

	// Import the models package or pass the table name
	db.Table("complains").Where("created-at >= ? AND created_at <= ?", startOfDay, endOfDay).Count(&count)

	// Increment count by 1 for the new record
	autoIncrement := count + 1

	// Format auto increment as 3-digit with leading zeros
	complainCode := fmt.Sprintf("%s%s%03d", datePrefix, userPrefix, autoIncrement)

	return complainCode
}
