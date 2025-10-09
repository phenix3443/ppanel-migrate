package main

import (
	"errors"
	"flag"
	"migrate/core/convert"
	"migrate/core/logger"
	ppanelModel "migrate/core/ppanel"
	"migrate/core/v2board"
	"migrate/internal/config"
	"time"

	"github.com/zeromicro/go-zero/core/conf"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// connectMySQL creates and returns a new MySQL database connection using the provided DSN.
func connectMySQL(dsn string) (*gorm.DB, error) {
	return gorm.Open(mysql.New(mysql.Config{DSN: dsn}), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 nil,
	})
}

var configFile = flag.String("f", "config.yaml", "the config file")

// main is the entry point for the migration tool. It loads configuration, connects to databases, and migrates users and plans from v2board to ppanel.
func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Validate required configuration
	if c.V2boardDataSource == "" || c.PPanelDataSource == "" {
		logger.Printf("Error", "Main", "Config", "V2BoardDataSource and PPanelDataSource must be specified in the config file.")
		return
	}
	if c.Migrate.MigrateAffiliate && !c.Migrate.MigrateAllUser {
		logger.Printf("Error", "Main", "Config", "MigrateAllUser must be true to migrate affiliate relationships.")
		return
	}

	// Connect to databases
	vdb, err := connectMySQL(c.V2boardDataSource)
	if err != nil {
		logger.Printf("Error", "Main", "DB", "Failed to connect to V2Board: %v", err)
		return
	}
	pdb, err := connectMySQL(c.PPanelDataSource)
	if err != nil {
		logger.Printf("Error", "Main", "DB", "Failed to connect to PPanel: %v", err)
		return
	}

	// Start transaction
	tx := pdb.Begin()

	// Get max user ID in ppanel
	var maxUserID int64
	if err := tx.Model(&ppanelModel.User{}).Select("MAX(id)").Scan(&maxUserID).Error; err != nil {
		logger.Printf("Error", "Main", "DB", "Failed to get max user ID in PPanel: %v", err)
		return
	}
	logger.Printf("Info", "Main", "StartID", "PPanel migration start user ID: %d", maxUserID)

	// Build plan ID mapping
	planMap := make(map[int64]int64, len(c.Migrate.Plans))
	for _, plan := range c.Migrate.Plans {
		planMap[plan.OldID] = plan.NewID
	}

	// Initialize migration converter
	ppanel := convert.NewConvert(vdb, tx,
		convert.WithPlanMap(planMap),
		convert.WithLongTermPlans(c.Migrate.LongTermPlanID),
		convert.WithAffiliate(c.Migrate.MigrateAffiliate),
		convert.WithAllUsers(c.Migrate.MigrateAllUser),
		convert.WithUnmatchedOnlyMigrateUser(c.Migrate.UnmatchedOnlyMigrateUser),
		convert.WithNeedOrder(c.Migrate.NeedOrder),
		convert.WithStartID(maxUserID),
	)

	// Fetch all users from v2board
	var users []*v2board.User
	if err := vdb.Model(&v2board.User{}).Find(&users).Error; err != nil {
		logger.Printf("Error", "Main", "DB", "Failed to fetch V2Board users: %v", err)
		return
	}

	// Migrate users
	for _, user := range users {
		if err := ppanel.Migrate(user); err != nil {
			if errors.Is(err, convert.ErrUserExists) {
				logger.Printf("Warning", "Main", "UserExists", "User %d (%s) already exists, skipping.", user.ID, user.Email)
				continue
			}
			tx.Rollback()
			logger.Printf("Error", "Main", "UserMigration", "Failed to migrate user %d (%s): %v", user.ID, user.Email, err)
			return
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		logger.Printf("Error", "Main", "Transaction", "Failed to commit: %v", err)
	} else {
		logger.Printf("Info", "Main", "Complete", "Total users processed: %d", len(users))
		logger.Printf("Info", "Main", "Stats", "%s", ppanel.GetStats(len(users)))
		logger.Printf("Info", "Main", "Complete", "Migration finished, transaction committed.")
	}

	// Ensure logs are flushed before exit
	time.Sleep(2 * time.Second)
}
