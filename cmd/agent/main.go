package main

import (
	"flag"
	"net"
	"os"
	"os/exec"
	"time"

	"code.google.com/p/go-uuid/uuid"

	"github.com/pivotal-cf/brokerapi/auth"
	"github.com/pivotal-cf/cf-redis-broker/agentconfig"
	"github.com/pivotal-cf/cf-redis-broker/api"
	"github.com/pivotal-cf/cf-redis-broker/availability"
	"github.com/pivotal-cf/cf-redis-broker/credentials"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-cf/cf-redis-broker/resetter"
	"github.com/pivotal-golang/lager"

	cf_lager "github.com/cloudfoundry-incubator/cf-lager"
	"github.com/codegangsta/negroni"
)

type portChecker struct{}

func (portChecker) Check(address *net.TCPAddr, timeout time.Duration) error {
	return availability.Check(address, timeout)
}

type commandRunner struct{}

func (commandRunner) Run(command *exec.Cmd) ([]byte, error) {
	return command.CombinedOutput()
}

func main() {

	configPath := flag.String("agentConfig", "", "Agent config yaml")
	logger := cf_lager.New("redis-agent")

	config, err := agentconfig.Load(*configPath)
	if err != nil {
		logger.Fatal("Error loading config file", err, lager.Data{
			"path": *configPath,
		})
	}

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

		newConfig.Set("requirepass", existingConf.Get("requirepass"))
	} else {
		newConfig.Set("requirepass", uuid.NewRandom().String())
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

	redisResetter := resetter.New(config.DefaultConfPath, config.ConfPath, new(portChecker), new(commandRunner), config.MonitExecutablePath)

	authWrapper := auth.NewWrapper(config.AuthConfiguration.Username, config.AuthConfiguration.Password)
	handler := authWrapper.Wrap(api.New(redisResetter, config.ConfPath, credentials.Parse))

	serverMiddleware := negroni.Classic()
	serverMiddleware.UseHandler(handler)
	serverMiddleware.Run("localhost:" + config.Port)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
