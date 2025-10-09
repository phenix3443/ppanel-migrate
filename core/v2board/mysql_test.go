package v2board

import (
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func connectMySQL(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: dsn,
	}), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func TestMySQL(t *testing.T) {

	t.Run("测试套餐查询", func(t *testing.T) {
		db, err := connectMySQL("root:mylove520@tcp(localhost:3306)/swift?charset=utf8mb4&parseTime=True&loc=Local")
		if err != nil {
			t.Logf("连接数据库失败: %v", err)
		}
		var plans []*V2BoardPlan
		err = db.Model(&V2BoardPlan{}).Find(&plans).Error
		if err != nil {
			t.Fatalf("查询数据失败: %v", err)
		}

		for _, plan := range plans {
			t.Logf("[套餐信息] ID: %d, 流量: %d GB, 月付价格: %d, 季付价格: %d, 半年付价格: %d, 年付价格: %d, 两年付价格: %d, 一次性价格: %d", plan.ID, plan.Transfer, plan.Month, plan.Quarter, plan.Half, plan.Year, plan.TwoYear, plan.OneTime)
		}
	})

	t.Run("测试用户查询", func(t *testing.T) {
		db, err := connectMySQL("root:mylove520@tcp(localhost:3306)/swift?charset=utf8mb4&parseTime=True&loc=Local")
		if err != nil {
			t.Logf("连接数据库失败: %v", err)
		}
		var users []*V2BoardUser
		err = db.Model(&V2BoardUser{}).Limit(20).Find(&users).Error
		if err != nil {
			t.Fatalf("查询数据失败: %v", err)
		}

		for _, user := range users {
			t.Logf("[用户信息] ID: %d, 邮箱: %s, 余额: %d, 套餐ID: %d, 流量配额: %d Bytes, 已上传流量: %d Bytes, 已下载流量: %d Bytes, 到期时间: %d, 最后使用时间: %d, 最后登录时间: %d", user.ID, user.Email, user.Balance, user.PlanID, user.Transfer, user.Upload, user.Download, user.ExpiredAt, user.LastUsedAt, user.LastLoginAt)
		}
	})

}
