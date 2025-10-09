# migrate

## 项目简介

`migrate` 是一个用 Go 语言编写的数据迁移工具，支持将 v2board 的用户、套餐、订单等数据迁移到 ppanel。适用于需要更换面板或整合数据的场景，支持灵活配置迁移规则。

## 目录结构

```
.
├── config.yaml              # 主配置文件
├── config.yaml.example      # 配置文件示例
├── main.go                  # 程序入口
├── core/                    # 核心迁移逻辑
│   ├── convert/             # 数据转换相关
│   ├── deduction/           # 扣费相关
│   ├── logger/              # 日志模块
│   ├── ppanel/              # ppanel 相关模型
│   ├── utils/               # 工具函数
│   └── v2board/             # v2board 相关模型
├── internal/                # 配置与业务逻辑
│   ├── config/              # 配置结构体
│   └── logic/               # 业务逻辑
├── logs/                    # 日志文件
└── migrate.log              # 迁移日志
```

## 安装与依赖

1. 安装 Go 1.24 及以上版本。
2. 克隆本项目并安装依赖：

```bash
git clone <your-repo-url>
cd migrate
go mod tidy
```

## 配置说明

请参考 `config.yaml.example`，复制为 `config.yaml` 并根据实际情况填写：

```yaml
V2boardDataSource: "root:123456@tcp(127.0.0.1:3306)/v2board?charset=utf8mb4&parseTime=True&loc=Local"
PPanelDataSource: "root:123456@tcp(127.0.0.1:3306)/ppanel?charset=utf8mb4&parseTime=True&loc=Local"
Migrate:
  Plans:
    - OldID: 3
      NewID: 110
    # 更多套餐映射...
  LongTermPlanID:
    - 6
    - 7
    - 8
  UnmatchedOnlyMigrateUser: false
  MigrateAllUser: true
  MigrateAffiliate: true
  NeedOrder: true
```

- `V2boardDataSource`/`PPanelDataSource`：MySQL 数据库连接串
- `Plans`：套餐 ID 映射
- 其他选项详见注释

## 使用方法

```bash
go run main.go -f config.yaml
```
或编译后运行：
```bash
go build -o migrate main.go
./migrate -f config.yaml
```

## 日志与错误

迁移日志默认输出到 `logs/migrate.log`，如遇错误请检查日志内容和配置项。

## 测试

如需测试迁移逻辑，可参考 `core/v2board/mysql_test.go`，或自行编写测试用例。

## 许可证

MIT License

