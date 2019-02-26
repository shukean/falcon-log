package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/shukean/falcon-log/common/config"
	"github.com/shukean/falcon-log/common/counter"
	"github.com/shukean/falcon-log/common/log"
	"github.com/streamrail/concurrent-map"
)

var (
	workers       chan bool
	counters      cmap.ConcurrentMap
	count_default float64 = 0
)

type itemType int8

const (
	kRule itemType = iota
	kAlive
)

type ruleItem struct {
	key  string
	num  float64
	rule config.Rule
}

type aliveItem struct {
	key    string
	times  int
	status float64
	alive  config.Alive
}

type counterItem struct {
	ItemType  itemType
	RuleItem  ruleItem
	AliveItem aliveItem
}

func watcherLog(filter *config.Filter) {
	log.Infof("filter file:%s, contain rules count:%d", filter.File, len(filter.Rules))
	for _, rule := range filter.Rules {
		if rule.SendType == config.SendTypeFalcon {
			item := counterItem{
				ItemType: kRule,
				RuleItem: ruleItem{key: rule.Key, num: count_default, rule: rule},
			}
			counters.Set(rule.Key, item)
		}
		log.Infof("filter file:%s, rule key:%s idx:%d (%s), include key:\"%s\", exclude key:\"%s\"",
			filter.File, rule.Key, rule.Index, rule.SendType, rule.Include, rule.Exclude)
	}
	if !filter.AliveCk.IsEmpty() {
		item := counterItem{
			ItemType:  kAlive,
			AliveItem: aliveItem{key: filter.Key, times: 0, status: 0, alive: filter.AliveCk},
		}
		counters.Set(filter.Key, item)
		log.Infof("filter file:%s, check alive, alive config:%v", filter.File, filter.AliveCk)
	}
	if !config.Cfg.Enabled {
		log.Infof("config enable is false, so stop monitor it")
		return
	}
	go func() {
		for line := range filter.Tail.Lines {
			log.Debugf("monitor log:%s", line.Text)
			checkLog(line.Text, filter)
			if !filter.AliveCk.IsEmpty() {
				if v, ok := counters.Get(filter.Key); ok {
					item := v.(counterItem)
					item.AliveItem.status += 1
					counters.Set(filter.Key, item)
				} else {
					log.Logger.Panicf("filter file:%s count msg not exists in alives", filter.File)
				}
			}
		}
	}()
}

func checkLog(content string, filter *config.Filter) {
	for _, rule := range filter.Rules {
		var matchs []string
		matchs = rule.RegexInclude.FindStringSubmatch(content)
		if matchs == nil || len(matchs) == 0 {
			log.Debugf("filter file:%s, rule idx:%d, include key:\"%s\" matched log content:%s failed", filter.File, rule.Index, rule.Include, content)
			continue
		}
		var val float64 = 1
		var err error
		if len(matchs) > 1 {
			if val, err = strconv.ParseFloat(matchs[1], 64); err != nil {
				log.Debugf("strconv failed, %s", matchs[1])
				val = 1
			}
		}
		log.Debugf("filter file:%s, rule idx:%d, include key:\"%s\" matched log content:%s successed, matchs:%v", filter.File, rule.Index, rule.Include, content, matchs)
		if rule.RegexExclude != nil {
			if rule.RegexExclude.MatchString(content) {
				log.Debugf("filter file:%s, rule idx:%d, exclude key:\"%s\"  matched log content(%s) successed", filter.File, rule.Index, rule.Exclude, content)
				continue
			}
		}
		if rule.SendType == config.SendTypeFalcon {
			if v, ok := counters.Get(rule.Key); ok {
				item := v.(counterItem)
				item.RuleItem.num += val
				counters.Set(rule.Key, item)
			} else {
				log.Logger.Panicf("rule key:%s count msg not exists in counter", rule.Key)
			}
		} else if rule.SendType == config.SendTypeCommand {
			argv := strings.Split(rule.Cmd, " ")
			argv = append(argv, content)
			cmd := exec.Command(argv[0], argv[1:]...)
			cmd.Env = append(os.Environ())
			out, err := cmd.Output()
			if err != nil {
				log.Fatalf("exec command:%s faild:%s, err:%s", rule.Cmd, out, err)
			} else {
				log.Infof("exec command:%s result:%s", rule.Cmd, out)
			}
		} else {
			log.Logger.Panicf("rule key:%s type:%s is out of range", rule.Key, rule.SendType)
		}
	}
}

func SplitCounterToFalcon(items []counterItem) {
	total := len(items)
	if total <= config.Cfg.Falcon.MaxBatchNum {
		PushToFalcon(items)
	} else {
		for i := 0; i < total; i += config.Cfg.Falcon.MaxBatchNum {
			end := i + config.Cfg.Falcon.MaxBatchNum
			if end <= total {
				tmp := items[i:end]
				PushToFalcon(tmp)
			} else {
				tmp := items[i:total]
				PushToFalcon(tmp)
			}
		}
	}
}

func PushToFalcon(data []counterItem) {
	msgs := make([]config.FalconAgentData, 0, len(data))
	for _, item := range data {
		var m config.Params
		var num float64
		var step int
		if item.ItemType == kRule {
			m = item.RuleItem.rule.Params
			num = item.RuleItem.num
			step = config.Cfg.Interval
		} else if item.ItemType == kAlive {
			m = item.AliveItem.alive.Params
			num = item.AliveItem.status
			step = config.Cfg.Interval * item.AliveItem.alive.MultiInterval
		} else {
			log.Fatal("counter type is unknow")
			continue
		}
		msg := config.FalconAgentData{
			Metric:      m.Metric,
			Endpoint:    config.Cfg.Host,
			Timestamp:   time.Now().Unix(),
			Value:       num,
			Step:        step,
			CounterType: m.Type,
			Tags:        strings.Join(m.Tags, ","),
		}
		msgs = append(msgs, msg)
	}
	bytes, err := json.Marshal(msgs)
	if err != nil {
		log.Errorf("push data:%s to json error %s", msgs, err)
		return
	}
	timeout := time.Duration(time.Second * time.Duration(config.Cfg.Falcon.Timeout))
	client := http.Client{
		Timeout: timeout,
	}
	log.Debugf("post data: %s", string(bytes))
	resp, err := client.Post(config.Cfg.Falcon.Url, "plain/text", strings.NewReader(string(bytes)))
	if err != nil {
		log.Errorf("post failed: %s", err)
		return
	}
	defer resp.Body.Close()
	bytes, _ = ioutil.ReadAll(resp.Body)
	log.Debug("falcon push response is", string(bytes))
}

func sendFalcon() {
	workers <- true
	go func() {
		if len(counters.Items()) != 0 {
			items := make([]counterItem, 0, len(counters.Items()))
			for k, v := range counters.Items() {
				item := v.(counterItem)
				if item.ItemType == kAlive {
					item.AliveItem.times += 1
					if item.AliveItem.times >= item.AliveItem.alive.MultiInterval {
						items = append(items, item)
						item.AliveItem.times = 0
						item.AliveItem.status = 0
					}
				} else if item.ItemType == kRule {
					items = append(items, item)
					item.RuleItem.num = count_default
				}
				counters.Set(k, item)
			}
			log.Debugf("start push msg to falcon, num:%d", len(items))
			SplitCounterToFalcon(items)
		}
		<-workers
	}()
}

func initTimer() {
	log.Debugf("push falcon timer interval %d", config.Cfg.Interval)
	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(int64(config.Cfg.Interval)))
		for range ticker.C {
			sendFalcon()
		}
	}()
}

func main() {
	log.Info("Start monitor logs")

	workers = make(chan bool, 4)
	runtime.GOMAXPROCS(runtime.NumCPU())

	counters = counter.NewConcurrentMap()

	log.SetDebug(config.Cfg.Debug)
	initTimer()

	defer func() {
		log.LogFp.Close()
		log.PidFp.Close()
	}()
	err := syscall.Flock(int(log.PidFp.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		log.Fatal("lock pid file failed, err:%s", err)
		os.Exit(2)
	}
	defer func() {
		syscall.Flock(int(log.PidFp.Fd()), syscall.LOCK_UN)
	}()
	pid := os.Getpid()
	if pid < 1 {
		log.Fatal("get pid failed")
		os.Exit(2)
	}
	log.PidFp.Truncate(0)
	log.PidFp.Write([]byte(strconv.Itoa(pid)))

	go func() {
		for idx, _ := range config.Cfg.Filters {
			watcherLog(&config.Cfg.Filters[idx])
		}
	}()

	select {}
}
