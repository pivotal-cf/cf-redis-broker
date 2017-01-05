package main

import (
	"flag"
	"net"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/brokerapi/auth"
	"github.com/pivotal-cf/cf-redis-broker/agentapi"
	"github.com/pivotal-cf/cf-redis-broker/agentconfig"
	"github.com/pivotal-cf/cf-redis-broker/availability"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-cf/cf-redis-broker/resetter"
)

type portChecker struct{}

func (portChecker) Check(address *net.TCPAddr, timeout time.Duration) error {
	return availability.Check(address, timeout)
}

func main() {
	configPath := flag.String("agentConfig", "", "Agent config yaml")
	flag.Parse()

	logger := lager.NewLogger("redis-agent")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))

	config, err := agentconfig.Load(*configPath)
	if err != nil {
		logger.Fatal("Error loading config file", err, lager.Data{
			"path": *configPath,
		})
	}

	templateRedisConf(config, logger)

	redisResetter := resetter.New(
		config.DefaultConfPath,
		config.ConfPath,
		portChecker{},
	)
	redisResetter.Monit.SetExecutable(config.MonitExecutablePath)

	handler := auth.NewWrapper(
		config.AuthConfiguration.Username,
		config.AuthConfiguration.Password,
	).Wrap(
		agentapi.New(redisResetter, config.ConfPath),
	)

	http.Handle("/", handler)
	logger.Fatal("http-listen", http.ListenAndServe("localhost:"+config.Port, nil))
}

func templateRedisConf(config *agentconfig.Config, logger lager.Logger) {
	newConfig, err := redisconf.Load(config.DefaultConfPath)
	if err != nil {
		logger.Fatal("Error loading default redis.conf", err, lager.Data{
			"path": config.DefaultConfPath,
		})
	}

	if fileExists(config.ConfPath) {
		existingConf, err := redisconf.Load(config.ConfPath)
		if err != nil {
			logger.Fatal("Error loading existing redis.conf", err, lager.Data{
				"path": config.ConfPath,
			})
		}
		err = newConfig.InitForDedicatedNode(existingConf.Password())
	} else {
		err = newConfig.InitForDedicatedNode()
	}

	if err != nil {
		logger.Fatal("Error initializing redis conf for dedicated node", err)
	}

	err = newConfig.Save(config.ConfPath)
	if err != nil {
		logger.Fatal("Error saving redis.conf", err, lager.Data{
			"path": config.ConfPath,
		})
	}

	logger.Info("Finished writing redis.conf", lager.Data{
		"path": config.ConfPath,
		"conf": newConfig,
	})
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
