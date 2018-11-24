package log

import (
    "os"
    "strconv"
    "log"
    "fmt"
    "strings"
)

const (
    logPath string = "./log/"
    logFile string = "app.log"
    pidFile string = "pid"

    fDEBUG string = "DEBUG"
    fINFO  string = "INFO"
    fERROR string = "ERROR"
    fFATAL string = "FATAL"
)

type ykLog struct {
    log    *log.Logger
    debug  bool
}

var (
    logger ykLog
)

func init() {
    if _, err := os.Stat(logPath); os.IsNotExist(err) {
        os.Mkdir(logPath, 0755)
    }
    fp, err := os.OpenFile(logPath + logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
    if err != nil {
        fmt.Printf("open file %s failed, %s", logFile, err)
        os.Exit(2)
    }
    pid := os.Getpid()
    if pid < 1 {
        fmt.Println("get pid failed")
        os.Exit(2)
    }
    fp2, err := os.OpenFile(logPath + pidFile, os.O_RDWR|os.O_CREATE, 0644)
    if err != nil {
        fmt.Printf("open file %s failed, %s", pidFile, err)
        os.Exit(2)
    }
    fp2.Write([]byte(strconv.Itoa(pid)))
    fp2.Close()

    logger.log = log.New(fp, "", log.LstdFlags|log.Lshortfile)
    logger.debug = true
}

func SetDebug(debug bool) {
    logger.debug = debug
}

func Logf(level string, format string, args ...interface{}) {
    if ! logger.debug && level == fDEBUG {
        return
    }
    logger.log.Printf(level + " " + format, args...)
}

func Debug(arg interface{}, args ...interface{}) {
    Logf(fDEBUG, fmt.Sprint(arg) + strings.Repeat(" %v", len(args)), args...)
}

func Debugf(format string, args ...interface{}) {
    Logf(fDEBUG, format, args...)
}

func Info(arg interface{}, args ...interface{}) {
    Logf(fINFO, fmt.Sprint(arg) + strings.Repeat(" %v", len(args)), args...)
}

func Infof(format string, args ...interface{}) {
    Logf(fINFO, format, args...)
}

func Error(arg interface{}, args ...interface{}) {
    Logf(fERROR, fmt.Sprint(arg) + strings.Repeat(" %v", len(args)), args...)
}

func Errorf(format string, args ...interface{}) {
    Logf(fERROR, format, args...)
}

func Fatal(arg interface{}, args ...interface{}) {
    Logf(fFATAL, fmt.Sprint(arg) + strings.Repeat(" %v", len(args)), args...)
}

func Fatalf(format string, args ...interface{}) {
    Logf(fFATAL, format, args...)
}




