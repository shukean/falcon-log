# falcon-log
log monitor and send msg to falcon v1 agent  
用于监控日志文件关键字，并统计次数后向 falocn agent 报告的工具  

## config 配置
具体配置参见 conf/cfg.json 文件格式  
**注意事项**  
1. 同一个file的规则需要写在一起,否则无法启动  
2. 不同文件但是相同的 metric 时,注意需要用tags区分  
3. 尽量去掉不必要的文件监控,以减少推送的数据  

```
"enabled": true,     目前该值没有作用
"debug": false,      开启 debug 日志级别，用于调试，线上建议关闭
"interval": 60,      采集周期，则间隔多少秒上 falcon 汇报一次
"hostname": "",      如果不配置，则自动获取机器名， 用于 falcon 汇报参数
"worker_nr": 4,      向 falcon 发送的排队数量
"watcher_type": "poll",         监控文件的方式，centos 建议为空， macos 调试使用 poll
"falcon":{
   "url":"http://127.0.0.1:1988/v1/push",              falcon agent 监听http 的端口
   "timeout": 20,                                       连接 falcon 的超时时间
   "max_batch_num": 10                                 一次推送最大合并发送规则数量, 默认为10
},
"load_extensions": true,                              加载扩容的规则
"filters" :[
  {
    "file": "/tmp/test.log",              需要监控的日志文件, 文件名唯一
    "exists": false,                      为 true 是表示文件需要先存在, 默认为 false, 可以不设置
    "alive": {                            文件探活, 检查日志是否有滚动, 可以不设置
      "multi_interval": 3,                推送间隔数,即采集周期个数
      "params": {
         "metric":"zk_alive",
         "type": "GAUGE",
         "tags":[],
         "value": {"count": 0}
      }
    },
    "rules": [
      {
      "index": 1,                         规则序号, 一个 filter 内需要唯一
      "include": "ERROR",                 配置字符，支持 go 正则, 不可为空
      "exclude": "a",                     匹配后，过滤字符，支持 go 正则，类似： grep -v, 可以为空
      "params":{
            "metric":"zk_error",          falcon 报警字段
            "type": "GAUGE",              falcon 报警值规则
            "tags": ["file=/var/log/zookeeper/zookeeper.log"],       falcon 报警附加 tags
            "value": {
                "count": 0                 这里扩展用。。。
            }
          }
      },
      {}, {}                              可以有多组 rule
   ],
  {}, {}, {}                              可以有多组 filter
]
```

#### 扩展规则
```
[
  {
    "file": "/tmp/test.log",              需要监控的日志文件, 文件名唯一
    "exists": false,                      为 true 是表示文件需要先存在, 默认为 false, 可以不设置
    "alive": {                            文件探活, 检查日志是否有滚动, 可以不设置
      "multi_interval": 3,                推送间隔数,即采集周期个数
      "params": {
         "metric":"zk_alive",
         "type": "GAUGE",
         "tags":[],
         "value": {"count": 0}
      }
    },
    "rules": []
  }
]

```
