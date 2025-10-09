package config

type Config struct {
	V2boardDataSource string // V2board 数据源
	PPanelDataSource  string // PPanel 数据源
	Migrate           struct {
		Plans []struct {
			OldID int64 // 旧套餐ID
			NewID int64 // 新套餐ID
		}
		LongTerm                 bool    `json:",default=true"`  // 是否进行长期订阅迁移, 不迁移则转换成余额
		LongTermPlanID           []int64 `json:",optional"`      // 长期订阅套餐ID列表, 如果不迁移长期套餐，必填
		UnmatchedOnlyMigrateUser bool    `json:",default=false"` // 未匹配到套餐时，是否只迁移用户不迁移订阅，否则不迁移该用户
		MigrateAllUser           bool    `json:",default=false"` // 是否迁移所有用户，包括已迁移过的用户
		MigrateAffiliate         bool    `json:",default=false"` // 是否迁移推广关系
		NeedOrder                bool    `json:",default=false"` // 是否有支付订单记录，如果没有则不迁移订阅
	}
}
