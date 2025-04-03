
## 简化大型环境中分布式监控多个服务端口的痛点

监控定义端口可参考conf.d/aiqa.yaml,用来描述所监控的端口,并提供metrics
对于一个节点有多个服务，并且有多套环境的情况下，此方法很容易，可以并结合dingding进行告警,并提供metrics

在Prometheus.yml添加一个job即可，好处则不需要写很多target在prometheus中，则可以更简化维护
```
# 多云底座服务端口监控(aiqa、aitest、csdev、csqa)
  - job_name: '多云底座服务端口监控(aiqa、aitest、csdev、csqa)'
    static_configs:
      - targets: ['10.102.98.31:8081']

```
在prometheus rules配置告警规则
```
# 定义关于端口告警的配置
groups:
- name: port_status_alerts
  rules:
  - alert: PortDown
    expr: port_status == 0
    for: 1m
    labels:
      severity: 严重
    annotations:
      summary: "告警 Port {{ $labels.port }} on host {{ $labels.host }} is down"
      description: "此端口 {{ $labels.port }} 在此主机上 {{ $labels.host }} 已经down掉，请检查故障原因."

```
```
# 定义关于端口告警的配置
groups:
- name: port_status_alerts
  rules:
  - alert: PortDown
    expr: port_status == 0
    for: 1m
    labels:
      severity: 严重
    annotations:
      summary: "告警 Port {{ $labels.port }} on host {{ $labels.host }} is down"
      description: "此端口 {{ $labels.port }} 在此主机上 {{ $labels.host }} 已经down掉，请检查故障原因."

接着在Grafana添加自定义的监控图标，添加过滤指定的promsql即可过滤对应的主机
```

grafana定义promesql制作图表监控
```
port_status{host="10.102.17.1"}
port_status{host="10.102.15.1"}
```

这样就可以在一个图标中监控多个服务的端口了，如果发现其中一个端口改掉，则对应的指标则会down


systemd方式当前使用不稳定，可以直接二进制nohup来启动
编译跨平台二进制，设置目标操作系统为Linux，目标架构为amd64
```
GOOS=linux GOARCH=amd64 go build -o port_exporter_linux_amd64 port_exporter.go
```
设置目标操作系统为Windows，目标架构为amd64
```
GOOS=windows GOARCH=amd64 go build -o port_exporter_windows_amd64.exe port_exporter.go
```
设置目标操作系统为Darwin (macOS)，目标架构为amd64
```
GOOS=darwin GOARCH=amd64 go build -o port_exporter_darwin_amd64 port_exporter.go
```
