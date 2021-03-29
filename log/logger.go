package log

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"strings"
	"time"
)

// NewLogger 创建自定义zap logger对象
func NewLogger(project, app, env, logfile, logLevel string, maxSize, maxAge, maxBackup int, compress, consoleEnable bool) {
	// 日志路径规范：/service/logs/app/prod/${PROJECT_NAME}/${APP_NAME}/(debug.log|info.log|warn.log|error.log)
	var debugLog, infoLog, warnLog, errLog string

	if env != "dev" {
		debugLog = strings.Join([]string{logfile, env, project, app, "debug.log"}, "/")
		infoLog = strings.Join([]string{logfile, env, project, app, "info.log"}, "/")
		warnLog = strings.Join([]string{logfile, env, project, app, "warn.log"}, "/")
		errLog = strings.Join([]string{logfile, env, project, app, "error.log"}, "/")
	} else {
		// 本地测试时，日志文件写到当前目录
		debugLog = fmt.Sprintf("./%s-%s-debug.log", project, app)
		infoLog = fmt.Sprintf("./%s-%s-info.log", project, app)
		warnLog = fmt.Sprintf("./%s-%s-warn.log", project, app)
		errLog = fmt.Sprintf("./%s-%s-err.log", project, app)
	}

	debugWriter := genZapWriter(debugLog, maxSize, maxAge, maxBackup, compress)
	infoWriter := genZapWriter(infoLog, maxSize, maxAge, maxBackup, compress)
	warnWriter := genZapWriter(warnLog, maxSize, maxAge, maxBackup, compress)
	errWriter := genZapWriter(errLog, maxSize, maxAge, maxBackup, compress)

	// 配置日志内容格式
	encoderConf := genEncoderConf()
	// 日志文件要求使用json格式输出
	jsonEncoder := zapcore.NewJSONEncoder(encoderConf)

	// 日志级别配置，不能直接写zap.DebugLevel、zap.InfoLevel等，否则在写error级别的log时，debug、info、warn也会写一份
	debugLevel := zap.LevelEnablerFunc(func(lv zapcore.Level) bool {
		return lv <= zap.DebugLevel
	})
	infoLevel := zap.LevelEnablerFunc(func(lv zapcore.Level) bool {
		return lv > zap.DebugLevel && lv <= zap.InfoLevel
	})
	warnLevel := zap.LevelEnablerFunc(func(lv zapcore.Level) bool {
		return lv > zap.InfoLevel && lv <= zap.WarnLevel
	})
	errLevel := zap.LevelEnablerFunc(func(lv zapcore.Level) bool {
		return lv >= zap.ErrorLevel
	})

	// 启用多个输出流，不同级别的日志写到不同的日志文件中
	writers := []zapcore.Core{
		zapcore.NewCore(jsonEncoder, debugWriter, debugLevel),
		zapcore.NewCore(jsonEncoder, infoWriter, infoLevel),
		zapcore.NewCore(jsonEncoder, warnWriter, warnLevel),
		zapcore.NewCore(jsonEncoder, errWriter, errLevel),
	}

	// 是否启用日志屏幕输出
	if consoleEnable {
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConf)
		level := genZapLevel(logLevel)
		consoleCore := zapcore.NewCore(consoleEncoder, zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)), level)
		writers = append(writers, consoleCore)
	}

	// 创建zap core对象
	core := zapcore.NewTee(writers...)

	// 创建zap logger对象，同时添加两个option：日志打印行号、error级别的日志打印堆栈信息
	zl := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	defer func() {
		_ = zl.Sync()
	}()

	// 为了方便使用，zap提供了两个全局Logger对象：
	// *zap.Logger，通过zap.L()调用，
	// *zap.SugaredLogger，通过zap.S()调用
	// 我们需要调用zap.ReplaceGlobals()方法把默认的全局Logger对象替换掉，替换成我们自定义的Logger对象，替换之后我们就可以全局调用自定义的Logger来记录日志
	zap.ReplaceGlobals(zl)
}

// genZapLevel 把配置文件中设定的日志级别，转换为zap log level对象
func genZapLevel(logLevel string) zapcore.Level {
	var zapLevel zapcore.Level
	switch logLevel {
	case "debug":
		zapLevel = zap.DebugLevel
	case "info":
		zapLevel = zap.InfoLevel
	case "warn":
		zapLevel = zap.WarnLevel
	case "error":
		zapLevel = zap.ErrorLevel
	default:
		zapLevel = zap.InfoLevel
	}
	return zapLevel
}

// genZapWriter 创建zap writer对象，用于创建zap core对象
// zap本身不支持日志切割，需要配合lumberjack一起使用
func genZapWriter(logfile string, maxSize, maxAge, maxBackup int, compress bool) zapcore.WriteSyncer {
	lumbLogger := &lumberjack.Logger{
		Filename:   logfile,
		MaxSize:    maxSize,
		MaxAge:     maxAge,
		MaxBackups: maxBackup,
		Compress:   compress,
	}
	return zapcore.AddSync(lumbLogger)
}

// genEncoderConf 生成EncoderConfig，用于配置日志格式
// 日志规范文档：http://wiki.baozun.com/pages/viewpage.action?pageId=22545583
func genEncoderConf() zapcore.EncoderConfig {
	encoderConf := zap.NewProductionEncoderConfig()
	encoderConf.EncodeTime = zapTimeEncoder               // 日志规范要求时间格式到毫秒
	encoderConf.TimeKey = "timestamp"                     // 时间戳的key使用timestamp
	encoderConf.MessageKey = "message"                    // 消息的key使用message
	encoderConf.EncodeLevel = zapcore.CapitalLevelEncoder // 日志规范要求日志级别为大写格式
	return encoderConf
}

// zapTimeEncoder 用于日志时间格式化
// 日志规范要求时间戳到毫秒级
func zapTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05:000"))
}

// 调用log.NewLogger()之后，就可以直接使用zap.L().Error()等方法进行写入日志，但是为了适应日常习惯，重新定义Error()等方法
func Error(msg string, err error) {
	zap.L().Error(msg, zap.Error(err))
}

func Warn(msg string, fields ...zap.Field) {
	zap.L().Warn(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	zap.L().Info(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	zap.L().Debug(msg, fields...)
}
