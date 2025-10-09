package logger

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"gopkg.in/natefinch/lumberjack.v2"
)

func init() {
	log.SetOutput(&lumberjack.Logger{
		Filename:   "logs/migrate.log", // 日志文件路径
		MaxSize:    100,                // 每个日志文件最大尺寸（MB）
		MaxBackups: 10,                 // 最多保留旧日志文件数量
		MaxAge:     30,                 // 最多保留旧日志文件天数
		Compress:   true,               // 是否压缩备份文件
	})
	log.SetFlags(log.LstdFlags)
}

// callerLocation 获取调用位置的相对路径和行号
func callerLocation(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown:0"
	}

	// 获取当前工作目录（项目根路径）
	wd, err := filepath.Abs(".")
	if err != nil {
		return fmt.Sprintf("%s:%d", file, line)
	}

	// 转换为相对路径
	rel, err := filepath.Rel(wd, file)
	if err != nil {
		return fmt.Sprintf("%s:%d", file, line)
	}
	return fmt.Sprintf("%s:%d", rel, line)
}

// Printf 格式化日志输出 - 时间戳 [级别] <模块> ｜ 事件 - 描述: 错误详情 - 调用位置
func Printf(level, module, event, desc string, args ...interface{}) {
	message := fmt.Sprintf(desc, args...)
	// 获取调用该日志函数的文件名和行号
	location := callerLocation(2) // 2 层：Printf <- 调用者
	log.Printf("[%s] <%s> %s | %s - %s", level, module, event, message, location)
}
