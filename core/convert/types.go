package convert

import "time"

var convertPlans = map[int64]int64{
	110: 110,
	210: 210,
	310: 310,
	410: 410,
}

type Subscribe struct {
	UserID      int64     // PPanel 用户ID
	SubscribeID int64     // PPanel 订阅ID
	StartAt     time.Time // 开始时间
	ExpiredAt   time.Time // 到期时间
}
