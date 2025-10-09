package convert

import (
	"errors"
	"fmt"
	"math/big"
	"migrate/core/logger"
	"strconv"
	"time"

	"migrate/core/ppanel"
	"migrate/core/v2board"

	"github.com/google/uuid"
	"github.com/perfect-panel/server/pkg/random"
	"github.com/perfect-panel/server/pkg/uuidx"
	"gorm.io/gorm"
)

var (
	ErrUserExists     = errors.New("user already exists")
	SubscribeNotFound = errors.New("subscription not found")
)

type Convert struct {
	v2boardDB                *gorm.DB
	ppanelDB                 *gorm.DB
	count                    int64
	balance                  int64
	order                    int64
	planMap                  map[int64]int64 // planMap: mapping from old plan IDs to new plan IDs
	longTerm                 bool            // longTerm: whether to migrate long-term plans
	longTermPlans            []int64         // longTermPlans: list of long-term plan IDs
	startID                  int64           // startID: starting user ID for PPanel
	needOrder                bool            // needOrder: whether payment order records are required for subscription migration
	allUsers                 bool            // allUsers: whether to migrate all users
	affiliate                bool            // affiliate: whether to migrate affiliate relationships
	unmatchedOnlyMigrateUser bool            // unmatchedOnlyMigrateUser: if true, only migrate users when subscriptions are unmatched (effective only when allUsers is false)
}

type Options func(*Convert)

// WithPlanMap sets the mapping of plan IDs
func WithPlanMap(planMap map[int64]int64) Options {
	return func(c *Convert) {
		c.planMap = planMap
	}
}

// WithLongTermPlans sets the list of long-term plan IDs
func WithLongTermPlans(plans []int64) Options {
	return func(c *Convert) {
		c.longTermPlans = plans
	}
}

// WithStartID sets the starting user ID for PPanel
func WithStartID(id int64) Options {
	return func(c *Convert) {
		c.startID = id
	}
}

// WithNeedOrder sets whether payment order records are required
func WithNeedOrder(need bool) Options {
	return func(c *Convert) {
		c.needOrder = need
	}
}

// WithAllUsers sets whether to migrate all users
func WithAllUsers(all bool) Options {
	return func(c *Convert) {
		c.allUsers = all
	}
}

// WithUnmatchedOnlyMigrateUser sets whether to migrate only users when subscriptions are unmatched
func WithUnmatchedOnlyMigrateUser(unmatched bool) Options {
	return func(c *Convert) {
		c.unmatchedOnlyMigrateUser = unmatched
	}
}

// WithAffiliate sets whether to migrate affiliate relationships
func WithAffiliate(aff bool) Options {
	return func(c *Convert) {
		c.affiliate = aff
	}
}

// NewConvert creates a new Convert instance with the provided database connection.
func NewConvert(v2, pp *gorm.DB, opts ...Options) *Convert {
	m := &Convert{
		v2boardDB: v2,
		ppanelDB:  pp,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Migrate starts migrating a user from v2board to ppanel
func (c *Convert) Migrate(old *v2board.User) error {
	if old.IsAdmin {
		logger.Printf("Info", "Migrate User", "Skip admin user", "User %d (%s) is admin, skipping migration", old.ID, old.Email)
		return nil
	}

	var err error
	var user *ppanel.User
	if c.allUsers {
		user, err = c.ConvertUser(old)
		if err != nil {
			logger.Printf("Error", "Migrate User", "Failed to migrate user", "Failed to migrate user %d (%s): %v", old.ID, old.Email, err)
			return err
		}
		logger.Printf("Info", "Migrate User", "User migrated successfully", "User %d (%s) migrated successfully, PPanel user ID: %d", old.ID, old.Email, user.Id)
	}

	if c.affiliate && old.InviteUserId != 0 {
		aff, e := c.FindAffiliate(old.InviteUserId)
		if e != nil {
			logger.Printf("Error", "Migrate User", "Failed to find referrer", "Failed to find referrer for user %d (%s): %v", old.ID, old.Email, e)
		} else {
			if aff != nil && user != nil {
				user.RefererId = aff.Id
				if err = c.ppanelDB.Save(user).Error; err != nil {
					logger.Printf("Error", "Migrate User", "Failed to set referrer", "Failed to set referrer for user %d (%s): %v", old.ID, old.Email, err)
				}
			}
		}
	}

	newSubID := c.findPlan(old.PlanID)
	if newSubID == 0 && old.ExpiredAt != 0 {
		if c.allUsers && user != nil {
			logger.Printf("Warning", "Migrate User", "Corresponding plan not found", "User %d (%s) plan ID %d not found in new platform (user:%d), skipping subscription migration", old.ID, old.Email, old.PlanID, user.Id)
		}
		logger.Printf("Info", "Migrate User", "Plan not found", "User %d (%s) plan ID %d not found in new platform, skipping migration", old.ID, old.Email, old.PlanID)
		return nil
	}
	if old.PlanID == 0 && c.needOrder {
		logger.Printf("Info", "Migrate User", "Skip user without plan", "User %d (%s) has no plan, skipping migration", old.ID, old.Email)
		return nil
	}
	var order v2board.Order
	err = c.v2boardDB.Model(&v2board.Order{}).Where("`user_id` = ? AND `plan_id` = ? AND `status` IN ? ", old.ID, old.PlanID, []int64{1, 3}).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Printf("Warning", "Migrate User", "Order not found", "User %d (%s) order not found, skipping migration", old.ID, old.Email)
			return nil
		}
		logger.Printf("Error", "Migrate User", "Order query failed", "Failed to query order for user %d (%s): %v, skipping migration", old.ID, old.Email, err.Error())
		return err
	}

	//expiredAt := time.Unix(old.ExpiredAt, 0)
	// Long term subscription without expiration date, convert to balance
	if old.ExpiredAt == 0 && c.isLongSubscribe(old.PlanID) {
		if order.ID == 0 {
			logger.Printf("Info", "Migrate User", "Skip long-term plan user without order", "User %d (%s) long-term plan order not found, skipping balance conversion", old.ID, old.Email)
			return nil
		}

		logger.Printf("Info", "Migrate User", "Preparing to migrate user", "User %d (%s) with long-term plan and no expiration date", old.ID, old.Email)
		if user == nil {
			user, err = c.ConvertUser(old)
			if err != nil {
				if errors.Is(err, ErrUserExists) {
					logger.Printf("Warning", "Migrate User", "User already exists", "User %d (%s) already exists, skipping migration", old.ID, old.Email)
					return nil
				}
				logger.Printf("Error", "Migrate User", "Migration error", "Error occurred while migrating user %d (%s): %v", old.ID, old.Email, err)
				return err
			}
		}

		amount := order.TotalAmount + order.BalanceAmount
		if amount <= 0 {
			logger.Printf("Warning", "Migrate User", "Order amount is zero", "User %d (%s) order amount is 0, skipping migration", old.ID, old.Email)
			return nil
		}
		// Query subscription info
		var plan *v2board.Plan
		err = c.v2boardDB.Model(&v2board.Plan{}).Where("`id` = ?", old.PlanID).First(&plan).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Printf("Warning", "Migrate User", "Plan not found", "User %d (%s) plan not found, skipping migration", old.ID, old.Email)
				return nil
			}
			logger.Printf("Error", "Migrate User", "Plan query failed", "Failed to query plan for user %d (%s): %v", old.ID, old.Email, err)
			return err
		}
		// Calculate traffic unit price
		if plan.OneTime == 0 {
			logger.Printf("Warning", "Migrate User", "Plan price is zero", "User %d (%s) plan price is 0, skipping migration", old.ID, old.Email)
			return nil
		}
		bigTransfer := new(big.Float)
		bigTransfer.SetInt64(old.Transfer) // Convert to bytes

		bigAmount := new(big.Float)
		bigAmount.SetInt64(order.TotalAmount + order.BalanceAmount)

		// Calculate traffic unit price
		trafficUnitPrice := new(big.Float).Quo(bigAmount, bigTransfer)

		// Calculate remaining traffic
		remainingTraffic := old.Transfer - old.Upload - old.Download
		bigRemaining := new(big.Float)
		bigRemaining.SetInt64(remainingTraffic)

		// Calculate remaining traffic amount
		remainingAmount := new(big.Float).Mul(bigRemaining, trafficUnitPrice)
		// Convert remaining traffic amount to integer
		remainingAmountInt, _ := remainingAmount.Int64()
		if remainingAmountInt <= 0 {
			logger.Printf("Warning", "Migrate User", "Remaining traffic amount is zero", "User %d (%s) remaining traffic amount is 0, plan traffic: %d Bytes, used traffic: %d Bytes", old.ID, old.Email, old.Transfer, old.Download+old.Upload)
			return nil
		}
		logger.Printf("Info", "Migrate User", "Long-term plan migration", "User %d (%s) long-term plan migration, plan traffic: %d Bytes, used traffic: %d Bytes, remaining traffic amount: %d, order total amount: %d", old.ID, old.Email, old.Transfer, old.Download+old.Upload, remainingAmountInt, amount)
		// Add balance
		user.Balance += remainingAmountInt
		if err = c.ppanelDB.Save(user).Error; err != nil {
			logger.Printf("Error", "Migrate User", "Balance update failed", "Failed to update balance for user %d (%s): %v", old.ID, old.Email, err.Error())
			return err // Failed to update balance
		}
		c.balance += remainingAmountInt
		c.order += amount
		logger.Printf("Info", "Migrate User", "Long-term plan migration successful", "User %d (%s) long-term plan migration successful, user balance: %d", old.ID, old.Email, remainingAmountInt)
		return nil

	}
	// Fixed term subscription with expiration date, convert to subscription
	if newSubID != 0 && old.ExpiredAt != 0 {
		// Query order info

		if order.ID == 0 && c.needOrder {
			logger.Printf("Info", "Migrate User", "Skip user without order", "User %d (%s) plan order not found, skipping subscription migration", old.ID, old.Email)
			return nil
		}

		amount := order.TotalAmount + order.BalanceAmount
		logger.Printf("Info", "Migrate User", "Preparing to migrate user", "User %d (%s) with fixed term plan, plan amount: %d", old.ID, old.Email, amount)
		c.order += amount

		if user == nil {
			user, err = c.ConvertUser(old)
			if err != nil {
				if errors.Is(err, ErrUserExists) {
					logger.Printf("Warning", "Migrate User", "User already exists", "User %d (%s) already exists, skipping migration", old.ID, old.Email)
					return nil
				}
				logger.Printf("Error", "Migrate User", "Migration error", "Error occurred while migrating user %d (%s): %v", old.ID, old.Email, err)
				return err
			}
		}

		// Transfer cycle subscription
		err = c.ConvertCycleSubscribe(&Subscribe{
			UserID:      user.Id,
			SubscribeID: newSubID,
			StartAt:     time.Unix(order.PaidAt, 0),
			ExpiredAt:   time.Unix(old.ExpiredAt, 0),
		})
		if err != nil {
			logger.Printf("Error", "Migrate User", "Cycle subscription transfer failed", "Failed to transfer cycle subscription for user %d (%s): %v", old.ID, old.Email, err.Error())
			return err
		}
		logger.Printf("Info", "Migrate User", "Cycle subscription migrated successfully", "User %d (%s) cycle subscription migrated successfully, new user ID: %d, new plan ID: %d, expiration date: %s, original plan amount: %d", old.ID, old.Email, user.Id, newSubID, time.Unix(old.ExpiredAt, 0).Format("2006-01-02 15:04:05"), amount)
		return nil
	}

	if c.unmatchedOnlyMigrateUser && user == nil {
		user, err = c.ConvertUser(old)
		if err != nil {
			if errors.Is(err, ErrUserExists) {
				logger.Printf("Warning", "Migrate User", "User already exists", "User %d (%s) already exists, skipping migration", old.ID, old.Email)
				return nil
			}
			logger.Printf("Error", "Migrate User", "Migration error", "Error occurred while migrating user %d (%s): %v", old.ID, old.Email, err)
			return err
		}
		logger.Printf("Info", "Migrate User", "Unmatched subscription", "User %d (%s) unmatched subscription, user migrated successfully, PPanel user ID: %d", old.ID, old.Email, user.Id)
		return nil
	}

	logger.Printf("Warning", "Migrate User", "Unmatched subscription migration", "User %d (%s) unmatched subscription migration | old plan ID %d | expiration date %s", old.ID, old.Email, old.PlanID, time.Unix(old.ExpiredAt, 0).Format("2006-01-02 15:04:05"))
	return nil
}

// ConvertUser migrates a user to the ppanel platform
func (c *Convert) ConvertUser(old *v2board.User) (*ppanel.User, error) {
	var user ppanel.User
	err := c.ppanelDB.Joins("JOIN user_auth_methods ON user_auth_methods.user_id = user.id").
		Where("user_auth_methods.auth_type = ? AND user_auth_methods.auth_identifier = ?", "email", old.Email).
		First(&user).Error
	if err == nil {
		// User already exists
		return &user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var algo = "bcrypt"
	if old.PasswordAlgo != "" {
		algo = old.PasswordAlgo
	}

	// Create user
	user = ppanel.User{
		Password: old.Password,
		Algo:     algo,
		Salt:     old.PasswordSalt,
		Avatar:   "",
		Balance:  old.Balance,
		Enable:   true,
	}
	if c.affiliate {
		user.Commission = old.CommissionBalance // Commission
	}

	if err := c.ppanelDB.Create(&user).Error; err != nil {
		return nil, err
	}
	// Set user invite code
	user.ReferCode = uuidx.UserInviteCode(user.Id)
	if err := c.ppanelDB.Save(&user).Error; err != nil {
		return nil, err
	}

	// Create user authentication method
	authMethod := &ppanel.AuthMethods{
		UserId:         user.Id,
		AuthType:       "email",
		AuthIdentifier: old.Email,
		Verified:       false,
	}
	if err := c.ppanelDB.Create(authMethod).Error; err != nil {
		return nil, err
	}
	c.count++
	c.balance += old.Balance
	return &user, nil
}

// ConvertCycleSubscribe migrates a cycle subscription to the ppanel platform
func (c *Convert) ConvertCycleSubscribe(sub *Subscribe) error {
	var info *ppanel.Subscribe
	err := c.ppanelDB.Model(&ppanel.Subscribe{}).Where("`id` = ?", sub.SubscribeID).First(&info).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SubscribeNotFound // Subscription not found
		}
		return err // Other errors
	}
	if info == nil {
		return SubscribeNotFound // Subscription not found
	}

	s := strconv.FormatInt(time.Now().UnixMicro(), 10) + random.Key(6, 1)
	// Add cycle subscription to user
	userSubscribe := &ppanel.UserSubscribe{
		UserId:      sub.UserID,
		User:        ppanel.User{},
		OrderId:     0,
		SubscribeId: info.Id,
		StartTime:   time.Now(),
		ExpireTime:  sub.ExpiredAt,
		FinishedAt:  nil,
		Traffic:     info.Traffic,
		Download:    0,
		Upload:      0,
		Token:       uuidx.SubscribeToken(s),
		UUID:        uuid.New().String(),
		Status:      1,
	}
	if err = c.ppanelDB.Create(userSubscribe).Error; err != nil {
		return err
	}

	return nil
}

// findPlan finds the new platform plan ID based on the old platform plan ID
func (c *Convert) findPlan(oldID int64) int64 {
	for old, id := range c.planMap {
		if old == oldID {
			return id
		}
	}
	return 0
}

// isLongSubscribe checks if the plan is a long-term plan
func (c *Convert) isLongSubscribe(old int64) bool {
	for _, id := range c.longTermPlans {
		if id == old {
			return true
		}
	}
	return false
}

// Count returns the number of migrated users
func (c *Convert) Count() int64 {
	return c.count
}

// GetStats returns migration statistics
func (c *Convert) GetStats(totalUsers int) string {
	return fmt.Sprintf("[Info] Migration stats: Total migrated users: %d, Total users processed: %d, Total order amount: %d, Total balance: %d", c.count, totalUsers, c.order, c.balance)
}

// FindAffiliate finds the referrer of a user in the new platform by email. If not found, migrates the referrer user.
func (c *Convert) FindAffiliate(old int64) (*ppanel.User, error) {
	var oldUser v2board.User
	err := c.v2boardDB.Model(&v2board.User{}).Where("`id` = ?", old).First(&oldUser).Error
	if err != nil {
		return nil, err
	}
	// Try to find the user by email in the new platform
	var auth ppanel.AuthMethods
	if err := c.ppanelDB.Model(&ppanel.AuthMethods{}).Where("`auth_type` = ? AND `auth_identifier` = ?", "email", oldUser.Email).First(&auth).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.ConvertUser(&oldUser)
		}
		return nil, err
	}
	if auth.UserId > 0 {
		var newUser ppanel.User
		if err := c.ppanelDB.Model(&ppanel.User{}).Where("`id` = ?", auth.UserId).First(&newUser).Error; err != nil {
			return nil, err
		}
		return &newUser, nil
	}

	return nil, nil
}
