package log

import (
    "os"
    "log"
    "fmt"
    "strings"
)

const (
    logPath string = "./log/"
    logFile string = "app.log"
    pidFile string = "pid"

    fDEBUG string = "[DEBUG]"
    fINFO  string = "[INFO]"
    fERROR string = "[ERROR]"
    fFATAL string = "[FATAL]"
)

type ykLog struct {
    log    *log.Logger
    debug  bool
}

var (
    Logger      ykLog
    LogFp       *os.File
    PidFp       *os.File
)

func init() {
    var err error 
    if _, err := os.Stat(logPath); os.IsNotExist(err) {
        os.Mkdir(logPath, 0755)
    }
    LogFp, err = os.OpenFile(logPath + logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
    if err != nil {
        fmt.Printf("open log file:%s failed, err:%s", logFile, err)
        os.Exit(2)
    }

    PidFp, err = os.OpenFile(logPath + pidFile, os.O_RDWR|os.O_CREATE, 0644)
    if err != nil {
        fmt.Printf("open pid file:%s failed, err:%s", pidFile, err)
        os.Exit(2)
    }

    Logger.log = log.New(LogFp, "", log.LstdFlags|log.Lshortfile)
    Logger.debug = true

}

func SetDebug(debug bool) {
    Logger.debug = debug
}

func Logf(level string, format string, v ...interface{}) {
    if ! Logger.debug && level == fDEBUG {
        return
    }
    Logger.log.Printf(level + format, v...)
}

func Debug(v ...interface{}) {
    Logf(fDEBUG, strings.Repeat(" %v", len(v)), v...)
}

func Debugf(format string, v ...interface{}) {
    Logf(fDEBUG, " " + format, v...)
}

func Info(v...interface{}) {
    Logf(fINFO, strings.Repeat(" %v", len(v)), v...)
}

func Infof(format string, v...interface{}) {
    Logf(fINFO, " " + format, v...)
}

func Error(v ...interface{}) {
    Logf(fERROR, strings.Repeat(" %v", len(v)), v...)
}

func Errorf(format string, v ...interface{}) {
    Logf(fERROR, " " + format, v ...)
}

func Fatal(v ...interface{}) {
    Logf(fFATAL, strings.Repeat(" %v", len(v)), v...)
}

func Fatalf(format string, v ...interface{}) {
    Logf(fFATAL, " " + format, v ...)
}

func (log *ykLog) Fatal(v ...interface{}) {
    Fatal(v...)
}

func (log *ykLog) Fatalf(format string, v ...interface{}) {
    Fatalf(format, v...)
}

func (log *ykLog) Fatalln(v ...interface{}) {
    Fatal(v...)
}

func (log *ykLog) Panic(v ...interface{}) {
    Fatal(v...)
    os.Exit(1)
}

func (log *ykLog) Panicf(format string, v ...interface{}) {
    Fatalf(format, v...)
    os.Exit(1)
}

func (log *ykLog) Panicln(v ...interface{}) {
    Fatal(v...)
    os.Exit(1)
}

func (log *ykLog) Print(v ...interface{}) {
    Info(v...);
}

func (log *ykLog) Printf(format string, v ...interface{}) {
    Infof(format, v...)
}

func (log *ykLog) Println(v ...interface{}) {
    Info(v...)
}
