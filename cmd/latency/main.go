package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/instance"
	"github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-cf/cf-redis-broker/redis/latency"
	"github.com/pivotal-golang/lager"
)

func main() {
	logger := lager.NewLogger("redis-latency-monitor")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	var configPath, latencyConfigPath string
	flag.StringVar(&configPath, "redisconf", "", "Path to Redis config file")
	flag.StringVar(&latencyConfigPath, "config", "", "Path to Latency config file")

	flag.Parse()

	if configPath == "" {
		logger.Fatal("find-config-file", errors.New("No Redis config file provided"), lager.Data{})
	}
	if latencyConfigPath == "" {
		logger.Fatal("find-latency-config-file", errors.New("No Latency config file provided"), lager.Data{})
	}

	redisConfigFinder := instance.NewRedisConfigFinder(filepath.Dir(configPath), filepath.Base(configPath))
	redisConfs, err := redisConfigFinder.Find()
	if err != nil {
		logger.Fatal("find-redis-config", err, lager.Data{})
	}

	redisConf := redisConfs[0].Conf

	redisClient, err := client.Connect(
		client.Port(redisConf.Port()),
	)
	if err != nil {
		logger.Fatal("redis-client-connect", err, lager.Data{})
	}

	latencyConf, err := latency.LoadConfig(latencyConfigPath)
	if err != nil {
		logger.Fatal("load-latency-config-file", err)
	}

	interval, _ := time.ParseDuration(latencyConf.Interval)

	monitor := latency.NewLatency(
		redisClient,
		latencyConf.LatencyFilePath,
		interval,
		logger,
	)

	logger.Info("Starting Latency Monitor")
	monitor.Start()

	for {
	}

}
