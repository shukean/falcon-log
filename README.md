# falcon-log
log monitor and send msg to falcon v1 agent  
由于监控日志文件关键字，并统计次数后行 falocn agent 报告的工具  

## config 配置
具体配置参见 conf/cfg.json 文件格式  

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
   "max_batch_num": 10                                 一次推送最大合并发送规则数量
   "timeout": 20                                       连接 falcon 的超时时间
},

"filters" :[
  {
    "file": "/tmp/test.log",              需要监控的日志文件名，不存在会报错
    "alive": {                            文件探活, 检查日志是否有滚动
      "multi_interval": 3,                推送间隔数,即多个个采集周期
      "params": {
         "metric":"zk_alive",
         "type": "GAUGE",
         "tags":[],
         "value": {"count": 0}
      }
    },
    "rules": [
      {
      "index": 1,                         规则序号
      "include": "ERROR",                 配置字符，支持 go 正则
      "exclude": "a",                     匹配后，过滤字符，支持 go 正则，类似： grep -v
      "params":{
            "metric":"zk_error",          falcon 报警字段
            "type": "GAUGE",              falcon 报警值规则
            "tags": ["file=/var/log/zookeeper/zookeeper.log"],       falcon 报警附加 tags
            "value": {
                "count": 0                 这里扩展用。。。
            }
          }
      },
      如果有多组 rule
   可以有多组 filter
```
