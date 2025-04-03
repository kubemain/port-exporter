package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// 定义一个指标来表示端口是否存活
var portStatus = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "port_status",
		Help: "Shows whether the port is up (1) or down (0)",
	},
	[]string{"host", "port", "label", "describe"},
)

// 初始化Prometheus注册表
func init() {
	prometheus.MustRegister(portStatus)
}

// 配置文件结构
type Config struct {
	Import   []string `yaml:"import"`
	Hosts    []Host   `yaml:"hosts"`
	LogLevel string   `yaml:"log_level"` // 添加日志级别字段
	Port     string   `yaml:"port"`
}

type Host struct {
	IP       string `yaml:"ip"`
	Describe string `yaml:"describe"`
	Ports    []Port `yaml:"ports"`
}

type Port struct {
	Port  string `yaml:"port"`
	Label string `yaml:"label"`
}

var logger *zap.Logger

// 初始化日志记录器
func initLogger(logLevel string) {
	var cfg zap.Config
	var err error

	switch logLevel {
	case "debug":
		cfg = zap.NewDevelopmentConfig()
	case "info":
		cfg = zap.NewProductionConfig()
		cfg.Level.SetLevel(zap.InfoLevel)
	case "error":
		cfg = zap.NewProductionConfig()
		cfg.Level.SetLevel(zap.ErrorLevel)
	case "off":
		cfg = zap.NewProductionConfig()
		cfg.Level.SetLevel(zap.FatalLevel)
	default:
		cfg = zap.NewProductionConfig()
		cfg.Level.SetLevel(zap.InfoLevel)
	}

	logger, err = cfg.Build()
	if err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}
}

// 检查端口是否存活
func checkPort(host Host, port Port) {
	address := net.JoinHostPort(host.IP, port.Port)
	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err == nil {
		conn.Close()
		portStatus.WithLabelValues(host.IP, port.Port, port.Label, host.Describe).Set(1)
		logger.Info("Port is up",
			zap.String("host", host.IP),
			zap.String("port", port.Port),
			zap.String("label", port.Label),
			zap.String("describe", host.Describe),
		)
	} else {
		portStatus.WithLabelValues(host.IP, port.Port, port.Label, host.Describe).Set(0)
		logger.Error("Port is down",
			zap.String("host", host.IP),
			zap.String("port", port.Port),
			zap.String("label", port.Label),
			zap.String("describe", host.Describe),
			zap.Error(err),
		)
	}
}

func main() {
	var configFile string

	// 创建一个新的cobra命令
	var rootCmd = &cobra.Command{
		Use:   "port_exporter",
		Short: "A simple port exporter for Prometheus",
		Long:  `port_exporter is a tool to monitor the status of specified ports and expose the metrics to Prometheus.`,
		Run: func(cmd *cobra.Command, args []string) {
			// 读取主配置文件
			config, err := loadConfig(configFile)
			if err != nil {
				fmt.Printf("Error loading config file: %v\n", err)
				os.Exit(1)
			}

			// 初始化日志记录器
			initLogger(config.LogLevel)

			// 启动一个goroutine来定期检查端口状态
			go func() {
				for {
					for _, host := range config.Hosts {
						for _, port := range host.Ports {
							go checkPort(host, port)
						}
					}
					time.Sleep(10 * time.Second)
				}
			}()

			// 启动HTTP服务器来暴露Prometheus指标
			http.Handle("/metrics", promhttp.Handler())
			logger.Info("Serving metrics on :" + config.Port)
			err = http.ListenAndServe(":"+config.Port, nil)
			if err != nil {
				logger.Fatal("Error starting HTTP server", zap.Error(err))
			}
		},
	}

	// 添加配置文件参数
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Path to the config file")
	rootCmd.MarkFlagRequired("config")

	// 执行cobra命令
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// 加载配置文件
func loadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	for _, importPath := range config.Import {
		err = importConfigs(&config, importPath)
		if err != nil {
			return nil, err
		}
	}

	return &config, nil
}

// 导入配置文件
func importConfigs(config *Config, importPath string) error {
	files, err := filepath.Glob(importPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		var importedConfig Config
		err = yaml.Unmarshal(data, &importedConfig)
		if err != nil {
			return err
		}

		config.Hosts = append(config.Hosts, importedConfig.Hosts...)
	}

	return nil
}
