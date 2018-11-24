package main

import (
    "runtime"
    "strconv"
    "time"
    "strings"
    "net/http"
    "io/ioutil"
    "encoding/json"

    "github.com/streamrail/concurrent-map"
    "github.com/shukean/falcon-log/common/log"
    "github.com/shukean/falcon-log/common/config"
    "github.com/shukean/falcon-log/common/counter"
)

var (
    workers chan bool
    counters cmap.ConcurrentMap
    keyToRules cmap.ConcurrentMap
    count_default float64 = 0
)

func watcherLog(filter *config.Filter) {
    log.Infof("filter file:%s, rule count:%d", filter.File, len(filter.Rules))
    for i, rule := range filter.Rules {
        counters.Set(rule.Key, count_default)
        keyToRules.Set(rule.Key, filter.Rules[i])
        log.Infof("filter file:%s, rule idx:%d, include:%s, exclude:%s", filter.File, rule.Index, rule.Include, rule.Exclude)
    }
    go func() {
        for line := range filter.Tail.Lines {
            log.Debugf("tail line: %s", line.Text)
            checkLog(line.Text, filter)
        }
    }()
}

func checkLog(content string, filter* config.Filter) {
    for _, rule := range filter.Rules {
        var matchs []string
        matchs = rule.RegexInclude.FindStringSubmatch(content)
        if matchs == nil || len(matchs) == 0 {
            log.Debugf("filter file:%s, rule idx:%d, matched incldue (%s) failed %v", filter.File, rule.Index, content, rule.Include)
            continue
        }
        var val float64 = 1
        var err error
        if len(matchs) > 1 {
            if val, err = strconv.ParseFloat(matchs[1], 64); err != nil {
                log.Debugf("strconv failed, %s", matchs[1]);
                continue
            }
        }
        log.Debugf("filter file:%s, rule idx:%d, matched include (%s), key:%s, matchs:%v", filter.File, rule.Index, content, rule.Include, matchs)
        if rule.RegexExclude != nil {
            if rule.RegexExclude.MatchString(content) {
                log.Debugf("filter file:%s, rule idx:%d, matched exclude (%s) failed %v", filter.File, rule.Index, content, rule.Exclude)
                continue
            }
        }
        if v, ok := counters.Get(rule.Key); ok {
            val2 := v.(float64)
            counters.Set(rule.Key, val2 + val)
        }
    }
}

func PushToFalcon(data map[string]float64) {
    msgs := make([]config.FalconAgentData, 0, len(data))
    for k, v := range data {
        r, _ := keyToRules.Get(k)
        m := r.(config.Rule).Params
        msg := config.FalconAgentData {
            Metric: m.Metric,
            Endpoint: config.Cfg.Host,
            Timestamp: time.Now().Unix(),
            Value: v,
            Step: config.Cfg.Interval,
            CounterType: m.Type,
            Tags: strings.Join(m.Tags, ","),
        }
        msgs = append(msgs, msg)
    }
    bytes, err := json.Marshal(msgs);
    if err != nil {
        log.Errorf("json error %s", err)
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
    log.Debug("rsp is", string(bytes))
}

func sendFalcon() {
    workers <- true
    go func() {
        if len(counters.Items()) != 0 {
            log.Debug("start push ...")
            data := make(map[string]float64, len(counters.Items()))
            for k, v := range counters.Items() {
                data[k] = v.(float64)
                counters.Set(k, count_default)
            }
            PushToFalcon(data)
        }
        <-workers
    }()
}

func initTimer() {
    log.Debugf("timer interval %d", config.Cfg.Interval)
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
    keyToRules = counter.NewConcurrentMap()

    log.SetDebug(config.Cfg.Debug)
    initTimer()

    go func() {
        for idx, _ := range config.Cfg.Filters {
            watcherLog(&config.Cfg.Filters[idx])
        }
    }()

    select {}
}
