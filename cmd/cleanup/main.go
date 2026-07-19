package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type User struct {
	Id         int            `gorm:"primaryKey"`
	Username   string         `gorm:"unique;index"`
	Role       int            `gorm:"type:int;default:1"`
	Email      string         `gorm:"index"`
	Quota      int            `gorm:"type:int;default:0"`
	PromoQuota int            `gorm:"type:int;default:0;column:promo_quota"`
	UsedQuota  int            `gorm:"type:int;default:0;column:used_quota"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (User) TableName() string { return "users" }

const RoleRootUser = 100

func tableExists(db *gorm.DB, name string) bool {
	var count int
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&count)
	return count > 0
}

func columnExists(db *gorm.DB, table, column string) bool {
	var count int
	db.Raw("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?", table, column).Scan(&count)
	return count > 0
}

func main() {
	sqlitePath := os.Getenv("SQLITE_PATH")
	if sqlitePath == "" {
		sqlitePath = "./data/new-api.db?_busy_timeout=30000"
	}
	pathOnly := sqlitePath
	if i := strings.Index(pathOnly, "?"); i >= 0 {
		pathOnly = pathOnly[:i]
	}
	if strings.HasPrefix(pathOnly, "file:") {
		pathOnly = strings.TrimPrefix(pathOnly, "file:")
	}
	dir := filepath.Dir(pathOnly)
	if dir != "" && dir != "." {
		os.MkdirAll(dir, 0o755)
	}
	fmt.Printf("Using database: %s\n", pathOnly)

	db, err := gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		fmt.Println("Failed to open database:", err)
		os.Exit(1)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	var tables []string
	db.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name").Scan(&tables)
	fmt.Println("\nTables in database:")
	for _, t := range tables {
		var count int64
		db.Raw("SELECT COUNT(*) FROM `" + t + "`").Scan(&count)
		fmt.Printf("  %s: %d rows\n", t, count)
	}
	fmt.Println()

	if !tableExists(db, "users") {
		fmt.Println("No 'users' table exists. Database is empty/fresh. Nothing to clean.")
		return
	}

	var rootUser User
	if err := db.Unscoped().Where("role = ?", RoleRootUser).First(&rootUser).Error; err != nil {
		fmt.Println("No root user found in database.")
		fmt.Println("Counting all users...")
		var allCount int64
		db.Model(&User{}).Unscoped().Count(&allCount)
		fmt.Printf("Total users (including soft-deleted): %d\n", allCount)
		if allCount > 0 {
			fmt.Println("\nNo root user found but users exist. Deleting ALL users to ensure clean state...")
			tx := db.Begin()
			for _, t := range tables {
				if columnExists(db, t, "user_id") {
					res := tx.Exec("DELETE FROM `" + t + "` WHERE user_id > 0")
					if res.Error == nil && res.RowsAffected > 0 {
						fmt.Printf("  Cleaned %s: %d rows\n", t, res.RowsAffected)
					}
				}
			}
			res := tx.Exec("DELETE FROM users WHERE id > 0")
			if res.Error == nil {
				fmt.Printf("  Deleted %d users\n", res.RowsAffected)
			}
			tx.Commit()
			fmt.Println("\nAll users deleted. Database is now clean. On first app start, root user will be created.")
		} else {
			fmt.Println("Database is already clean (no users).")
		}
		return
	}
	fmt.Printf("Root user preserved: id=%d, username=%s\n\n", rootUser.Id, rootUser.Username)

	var usersToDelete []User
	db.Unscoped().Where("id != ? AND role != ?", rootUser.Id, RoleRootUser).Find(&usersToDelete)
	fmt.Printf("Found %d non-root users to delete (including soft-deleted):\n", len(usersToDelete))
	if len(usersToDelete) == 0 {
		var allUsers []User
		db.Unscoped().Find(&allUsers)
		fmt.Printf("Database is clean - %d total user(s) (root only).\n", len(allUsers))
		return
	}

	var ids []int
	for _, u := range usersToDelete {
		ids = append(ids, u.Id)
		status := "active"
		if u.DeletedAt.Valid {
			status = "soft-deleted"
		}
		fmt.Printf("  - id=%d, username=%s, email=%s, quota=%d, promo_quota=%d, status=%s\n",
			u.Id, u.Username, u.Email, u.Quota, u.PromoQuota, status)
	}
	fmt.Println()

	tx := db.Begin()
	if tx.Error != nil {
		fmt.Println("Failed to begin transaction:", tx.Error)
		os.Exit(1)
	}

	totalDeleted := int64(0)

	for _, t := range tables {
		if t == "users" {
			continue
		}
		if !columnExists(db, t, "user_id") {
			continue
		}
		result := tx.Exec("DELETE FROM `"+t+"` WHERE user_id IN ?", ids)
		if result.Error != nil {
			fmt.Printf("  Warning: failed to clean %s: %v\n", t, result.Error)
		} else if result.RowsAffected > 0 {
			fmt.Printf("  Cleaned %s: %d rows\n", t, result.RowsAffected)
			totalDeleted += result.RowsAffected
		}
	}

	if tableExists(db, "redemptions") && columnExists(db, "redemptions", "used_user_id") {
		result := tx.Exec("DELETE FROM redemptions WHERE used_user_id IN ?", ids)
		if result.Error == nil && result.RowsAffected > 0 {
			fmt.Printf("  Cleaned redemptions (used_user_id): %d rows\n", result.RowsAffected)
			totalDeleted += result.RowsAffected
		}
	}

	result := tx.Unscoped().Where("id IN ?", ids).Delete(&User{})
	if result.Error != nil {
		tx.Rollback()
		fmt.Println("Failed to delete users:", result.Error)
		os.Exit(1)
	}
	fmt.Printf("\n  Deleted %d users from users table\n", result.RowsAffected)
	totalDeleted += result.RowsAffected

	if err := tx.Commit().Error; err != nil {
		fmt.Println("Failed to commit transaction:", err)
		os.Exit(1)
	}

	fmt.Printf("\nTotal rows cleaned: %d\n\n", totalDeleted)

	var remaining []User
	db.Unscoped().Find(&remaining)
	fmt.Printf("Remaining users: %d\n", len(remaining))
	for _, u := range remaining {
		fmt.Printf("  - id=%d, username=%s, role=%d, quota=%d, promo_quota=%d\n",
			u.Id, u.Username, u.Role, u.Quota, u.PromoQuota)
	}

	fmt.Println("\n=== Cleanup complete! ===")
	fmt.Println("All non-root users and their associated wallet/balance data have been hard-deleted.")
	fmt.Println("You can now sign up with your email and receive the signup credit.")
	fmt.Println("")
	fmt.Println("IMPORTANT: If Redis is running, the app may still serve cached user data.")
	fmt.Println("Run 'redis-cli FLUSHALL' to clear the cache, or restart the application.")
}
