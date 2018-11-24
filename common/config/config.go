package config

import (
    "encoding/json"
    "io/ioutil"
    "fmt"
    "os"
    "regexp"
    "errors"

    "github.com/hpcloud/tail"
    "github.com/shukean/falcon-log/common/log"
)

type Falcon struct {
    Url     string
    Timeout int
}

type Value struct {
    Count   int             `json:"count"`
}

type Params struct {
    Metric      string      `json:"metric"`
    Type        string      `json:"type"`
    Tags        []string    `json:"tags"`
    Value       Value       `json:"value"`
}

type Rule struct {
    Index     int           `json:"index"`
    Include   string        `json:"include"`
    Exclude   string        `json:"exclude"`
    Params    Params        `json:"params"`
    RegexInclude  *regexp.Regexp
    RegexExclude  *regexp.Regexp
    Key       string
    File      *string
}

type Filter struct {
    File    string              `json:"file"`
    Rules   []Rule              `json:"rules"`
    Tail    *tail.Tail
}

type Config struct {
    Enabled       bool            `json:"enabled"`
    Debug         bool            `json:"debug"`
    Interval      int             `json:"interval"`
    Host          string          `json:"hostname"`
    Falcon        Falcon          `json:"falcon"`
    Filters       []Filter        `json:"filters"`
    WorkerNr      int             `json:"worker_nr"`        // send falcon worker nr
    WatcherType   string          `json:"watcher_type"`     // poll or inotify
}

type FalconAgentData struct {
    Metric    string  `json:"metric"`    //统计纬度
    Endpoint  string  `json:"endpoint"`  //主机
    Timestamp int64   `json:"timestamp"` //unix时间戳,秒
    Value     float64 `json:"value"`     // 代表该metric在当前时间点的值
    Step      int     `json:"step"`      //  表示该数据采集项的汇报周期，这对于后续的配置监控策略很重要，必须明确指定。
    //COUNTER：指标在存储和展现的时候，会被计算为speed，即（当前值 - 上次值）/ 时间间隔
    //COUNTER：指标在存储和展现的时候，会被计算为speed，即（当前值 - 上次值）/ 时间间隔

    CounterType string `json:"counterType"` //只能是COUNTER或者GAUGE二选一，前者表示该数据采集项为计时器类型，后者表示其为原值 (注意大小写)
    //GAUGE：即用户上传什么样的值，就原封不动的存储
    //COUNTER：指标在存储和展现的时候，会被计算为speed，即（当前值 - 上次值）/ 时间间隔
    Tags string `json:"tags"` //一组逗号分割的键值对, 对metric进一步描述和细化, 可以是空字符串. 比如idc=lg，比如service=xbox等，多个tag之间用逗号分割
}


const configFile = "./conf/cfg.json"

var (
    Cfg     *Config
)

func CheckConfig(config *Config) error {
    var err error
    if config.Host == "" {
        if config.Host, err = os.Hostname(); err != nil {
            return err
        }
    }
    for i, f := range config.Filters {
        if _, err = os.Stat(f.File); err != nil {
            log.Fatalf("filter: %d file not exist %s", f.File, i)
            return err
        }
        for j, r := range f.Rules {
            if r.Include == "" {
                return errors.New("rule include is empty")
            }
            if config.Filters[i].Rules[j].RegexInclude, err = regexp.Compile(r.Include); err != nil {
                return err
            }
            if r.Exclude != "" {
                if config.Filters[i].Rules[j].RegexExclude, err = regexp.Compile(r.Exclude); err != nil {
                    return err
                }
            }
            config.Filters[i].Rules[j].Key = fmt.Sprintf("k%d-%d", i, r.Index)
        }
        if err = setTail(&config.Filters[i], config.WatcherType); err != nil {
            return err
        }
    }

    // todo
    return nil
}

func setTail(filter *Filter, watcher_type string) error {
    is_poll := false
    if watcher_type == "poll" {
        is_poll = true
    }
    tail, err := tail.TailFile(filter.File, tail.Config{Follow: true, ReOpen: true, MustExist: false, Poll: is_poll})
	if err != nil {
        return err
	}
    filter.Tail = tail
    return nil
}


func ReadConfig(file string) (*Config, error) {
    bytes, err := ioutil.ReadFile(configFile)
    if err != nil {
        return nil, err
    }
    var config *Config;
    if err := json.Unmarshal(bytes, &config); err != nil {
        log.Fatalf("json Unmarshal failed, ", err)
        return nil, err
    }

    if err := CheckConfig(config); err != nil {
        log.Fatalf("check config failed ", err)
        return nil, err
    }

    log.Infof("config init success, start to work ...")
    return config, nil
}

func init() {
    var err error
    Cfg, err = ReadConfig(configFile)
    if err != nil {
        log.Errorf("ERROR: read config failed")
        os.Exit(2)
    }
}
