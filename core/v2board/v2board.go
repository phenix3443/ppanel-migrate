package v2board

type Plan struct {
	ID       int64 `gorm:"column:id;primaryKey;autoIncrement"`
	Transfer int64 `gorm:"column:transfer_enable;default:0;comment:套餐流量，单位为GB"`
	Month    int64 `gorm:"column:month_price;default:0;comment:月付价格"`
	Quarter  int64 `gorm:"column:quarter_price;default:0;comment:季付价格"`
	Half     int64 `gorm:"column:half_year_price;default:0;comment:半年付价格"`
	Year     int64 `gorm:"column:year_price;default:0;comment:年付价格"`
	TwoYear  int64 `gorm:"column:two_year_price;default:0;comment:两年付价格"`
	OneTime  int64 `gorm:"column:onetime_price;default:0;comment:一次性价格"`
}

func (Plan) TableName() string {
	return "v2_plan"
}

type User struct {
	ID                int64  `gorm:"column:id;primaryKey;autoIncrement"`
	InviteUserId      int64  `gorm:"column:invite_user_id;default:NULL;comment:邀请人用户ID"`
	Password          string `gorm:"column:password;type:varchar(100);not null;default:'';comment:用户密码"`
	PasswordAlgo      string `gorm:"column:password_algo;type:varchar(20);default:'';comment:加密算法"`
	PasswordSalt      string `gorm:"column:password_salt;type:varchar(20);default:NULL;comment:密码盐值"`
	Email             string `gorm:"column:email;type:varchar(255);not null;unique;comment:用户邮箱"`
	Balance           int64  `gorm:"column:balance;default:0;comment:用户余额"`
	IsAdmin           bool   `gorm:"column:is_admin;default:false;comment:是否为管理员"`
	PlanID            int64  `gorm:"column:plan_id;default:0;comment:用户套餐ID"`
	Transfer          int64  `gorm:"column:transfer_enable;default:0;comment:用户流量配额，单位为Bytes"`
	Upload            int64  `gorm:"column:u;default:0;comment:用户已上传流量，单位为Bytes"`
	Download          int64  `gorm:"column:d;default:0;comment:用户已下载流量，单位为Bytes"`
	CommissionBalance int64  `gorm:"column:commission_balance;default:0;comment:用户推广佣金余额"`
	ExpiredAt         int64  `gorm:"column:expired_at;default:0;comment:用户到期时间，Unix时间戳"`
	LastUsedAt        int64  `gorm:"column:t;default:0;comment:用户最后使用时间，Unix时间戳"`
	LastLoginAt       int64  `gorm:"column:last_login_at;default:0;comment:用户最后登录时间，Unix时间戳"`
}

func (User) TableName() string {
	return "v2_user"
}

type Order struct {
	ID            int64  `gorm:"column:id;primaryKey;autoIncrement"`
	PlanID        int64  `gorm:"column:plan_id;default:0;comment:订单套餐ID"`
	UserID        int64  `gorm:"column:user_id;default:0;comment:订单用户ID"`
	Type          int64  `gorm:"column:type;default:0;comment:订单类型，1:新购 2:续费 3:升级"`
	Period        string `gorm:"column:period;type:varchar(255);not null;default:'';comment:订单周期"`
	TotalAmount   int64  `gorm:"column:total_amount;default:0;comment:订单总金额，单位为分"`
	BalanceAmount int64  `gorm:"column:balance_amount;default:0;comment:订单使用余额支付金额，单位为分"`
	User          int64  `gorm:"column:status;default:0;comment:订单状态，0:待支付 1:已支付 2:已取消"`
	PaidAt        int64  `gorm:"column:paid_at;default:0;comment:订单支付时间，Unix时间戳"`
}

func (Order) TableName() string {
	return "v2_order"
}
