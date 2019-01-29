package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/hpcloud/tail"
	"github.com/shukean/falcon-log/common/log"
)

type Falcon struct {
	Url         string `json:"url"`
	Timeout     int    `json:"timeout"`
	MaxBatchNum int    `json:"max_batch_num"`
}

type Value struct {
	Count int `json:"count"`
}

type Params struct {
	Metric string   `json:"metric"`
	Type   string   `json:"type"`
	Tags   []string `json:"tags"`
	Value  Value    `json:"value"`
}

type Alive struct {
	MultiInterval int    `json:"multi_interval"`
	Params        Params `json:"params"`
}

type Rule struct {
	Index        int    `json:"index"`
	Include      string `json:"include"`
	Exclude      string `json:"exclude"`
	Params       Params `json:"params"`
	RegexInclude *regexp.Regexp
	RegexExclude *regexp.Regexp
	Key          string
	File         *string
}

type Filter struct {
	File    string `json:"file"`
	Exists  bool   `json:"exists"`
	AliveCk Alive  `json:"alive"`
	Rules   []Rule `json:"rules"`
	Tail    *tail.Tail
	Key     string
}

type Config struct {
	Enabled        bool     `json:"enabled"`
	Debug          bool     `json:"debug"`
	Interval       int      `json:"interval"`
	Host           string   `json:"hostname"`
	Falcon         Falcon   `json:"falcon"`
	LoadExtentions bool     `json:"load_extensions"`
	Filters        []Filter `json:"filters"`
	WorkerNr       int      `json:"worker_nr"`    // send falcon worker nr
	WatcherType    string   `json:"watcher_type"` // poll or inotify
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

func (params Params) IsEmpty() bool {
	return params.Metric == "" || params.Type == ""
}

func (alive Alive) IsEmpty() bool {
	return alive.MultiInterval < 0 || alive.Params.IsEmpty()
}

func (rule Rule) IsEmpty() bool {
	return rule.Index < 0 || rule.Include == "" || rule.Params.IsEmpty()
}

func (filter Filter) IsEmpty() bool {
	return filter.File == ""
}

func (falcon Falcon) IsEmpty() bool {
	return falcon.Url == ""
}

const configDir = "./conf"
const configFile = "cfg.json"
const maxFalconPushBatchNum = 10

var (
	Cfg *Config
)

func CheckConfig(config *Config) error {
	var err error
	if config.Host == "" {
		if config.Host, err = os.Hostname(); err != nil {
			return err
		}
	}
	var cfgs = make(map[string]int)
	for i, f := range config.Filters {
		if f.IsEmpty() {
			log.Logger.Panicf("filter file is empty")
		}
		if _, err = os.Stat(f.File); err != nil {
			log.Infof("filter:%d monitor file:%s not exists", i, f.File)
			if f.Exists {
				return err
			}
		}
		for j, r := range f.Rules {
			if r.IsEmpty() {
				return fmt.Errorf("fileter:%s rule:%d check failed", f.File, r.Index)
			}
			if config.Filters[i].Rules[j].RegexInclude, err = regexp.Compile(r.Include); err != nil {
				return err
			}
			if r.Exclude != "" {
				if config.Filters[i].Rules[j].RegexExclude, err = regexp.Compile(r.Exclude); err != nil {
					return err
				}
			}
			config.Filters[i].Rules[j].Key = fmt.Sprintf("rk%d-%d", i, r.Index)
		}
		if err = setTail(&config.Filters[i], config.WatcherType); err != nil {
			return err
		}
		config.Filters[i].Key = fmt.Sprintf("fk%d", i)

		if _, ok := cfgs[f.File]; !ok {
			cfgs[f.File] = 0
		} else {
			cfgs[f.File] += 1
		}
	}
	for cfg_name, cfg_nr := range cfgs {
		if cfg_nr > 0 {
			return fmt.Errorf("monitor file:%s appear in multiple config files", cfg_name)
		}
	}
	if config.Falcon.MaxBatchNum <= 0 {
		config.Falcon.MaxBatchNum = maxFalconPushBatchNum
	}
	log.Infof("falcon config:%v, files nr:%d", config.Falcon, len(config.Filters))
	// todo
	return nil
}

func setTail(filter *Filter, watcher_type string) error {
	seek := tail.SeekInfo{
		Offset: 0,
		Whence: os.SEEK_END,
	}
	finfo, err := os.Stat(filter.File)
	if err == nil {
		if finfo.Size() > 0 {
			seek.Offset = finfo.Size()
		}
		seek.Whence = os.SEEK_SET
	} else {
		log.Fatalf("stat file:%s failed, err:%v", filter.File, err)
	}
	log.Infof("tail of file:%s set offset:%d", filter.File, seek.Offset)
	cfg := tail.Config{
		Follow:    true,
		ReOpen:    true,
		MustExist: false,
		Poll:      watcher_type == "poll",
		Location:  &seek,
		Logger:    &log.Logger,
	}
	tail, err := tail.TailFile(filter.File, cfg)
	if err != nil {
		return err
	}
	filter.Tail = tail
	return nil
}

func ReadConfig(file string) (*Config, error) {
	log.Infof("load base config:%s", file)
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var config *Config
	if err := json.Unmarshal(bytes, &config); err != nil {
		log.Fatalf("json Unmarshal failed, err:%s", err)
		return nil, err
	}
	return config, nil
}

func loadExtentions(dir string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.Name() == configFile {
			continue
		}
		log.Infof("load config extentions:%s", f.Name())
		file := fmt.Sprintf("%s/%s", configDir, f.Name())
		bytes, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		var extfilters []Filter
		if err := json.Unmarshal(bytes, &extfilters); err != nil {
			return err
		}
		for _, filter := range extfilters {
			//check
			Cfg.Filters = append(Cfg.Filters, filter)
		}
	}
	return nil
}

func init() {
	var err error
	var base_cfg_file = fmt.Sprintf("%s/%s", configDir, configFile)
	Cfg, err = ReadConfig(base_cfg_file)
	if err != nil {
		log.Fatalf("read config failed:%s", err)
		os.Exit(2)
	}
	if Cfg.LoadExtentions {
		err = loadExtentions(configDir)
		if err != nil {
			log.Fatalf("load extenstions failed:%s", err)
			os.Exit(2)
		}
	}

	if err := CheckConfig(Cfg); err != nil {
		log.Fatalf("check config failed, err:%s ", err)
		os.Exit(2)
	}

	log.Infof("config init success, start to work ...")
}
