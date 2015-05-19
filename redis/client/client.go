package client

import (
	"errors"
	"fmt"
	"strings"
	"time"

	redisclient "github.com/garyburd/redigo/redis"
	"github.com/pivotal-cf/cf-redis-broker/log"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-golang/lager"
)

func Connect(host string, conf redisconf.Conf) (Client, error) {
	address := fmt.Sprintf("%v:%v", host, conf.Get("port"))
	connection, err := redisclient.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	password := conf.Get("requirepass")
	if password != "" {
		if _, err := connection.Do("AUTH", password); err != nil {
			connection.Close()
			return nil, err
		}
	}

	return &client{conf: conf, connection: connection}, err
}

type Client interface {
	CreateSnapshot(int) error
	WaitUntilRedisNotLoading(timeoutMilliseconds int) error
	EnableAOF() error
	LastRDBSaveTime() (int64, error)
	InfoField(fieldName string) (string, error)
	GetConfig(key string) (string, error)
}

type client struct {
	conf       redisconf.Conf
	connection redisclient.Conn
}

func (client *client) WaitUntilRedisNotLoading(timeoutMilliseconds int) error {
	for i := 0; i < timeoutMilliseconds; i += 100 {
		loading, err := client.InfoField("loading")
		if err != nil {
			return err
		}

		if loading == "0" {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func (client *client) CreateSnapshot(timeoutInSeconds int) error {
	log.Logger().Info("redis_client", lager.Data{
		"event":   "creating_snapshot",
		"timeout": timeoutInSeconds,
	})

	lastSaveTime, err := client.LastRDBSaveTime()
	if err != nil {
		log.Logger().Error("redis_client", err, lager.Data{
			"event": "last_rdb_save_time",
		})
		return err
	}

	// sleep for a second to ensure unique timestamp for bgsave
	time.Sleep(time.Second)

	err = client.runBGSave()
	if err != nil {
		log.Logger().Error("redis_client", err, lager.Data{
			"event": "run_bg_save",
		})
		return err
	}

	err = client.waitForNewSaveSince(lastSaveTime, timeoutInSeconds)
	if err != nil {
		log.Logger().Error("redis_client", err, lager.Data{
			"event":            "wait_for_new_save_since",
			"last_time_save":   lastSaveTime,
			"time_out_seconds": timeoutInSeconds,
		})
		return err
	}

	log.Logger().Info("redis_client", lager.Data{
		"event": "creating_snapshot_done",
	})

	return nil
}

func (client *client) EnableAOF() error {
	return client.setConfig("appendonly", "yes")
}

func (client *client) runBGSave() error {
	bgSaveCommand := client.conf.CommandAlias("BGSAVE")
	_, err := client.connection.Do(bgSaveCommand)
	return err
}

func (client *client) LastRDBSaveTime() (int64, error) {
	saveTimeStr, err := client.connection.Do("LASTSAVE")
	if err != nil {
		return 0, err
	}

	return saveTimeStr.(int64), nil
}

func (client *client) InfoField(fieldName string) (string, error) {
	info, err := client.info()
	if err != nil {
		return "", fmt.Errorf("Error during redis info: %s" + err.Error())
	}

	value, ok := info[fieldName]
	if !ok {
		return "", errors.New(fmt.Sprintf("Unknown field: %s", fieldName))
	}

	return value, nil
}

func (client *client) waitForNewSaveSince(lastSaveTime int64, timeoutInSeconds int) error {
	for i := 0; i < timeoutInSeconds; i++ {
		latestSaveTime, err := client.LastRDBSaveTime()
		if err != nil {
			return err
		}

		if latestSaveTime > lastSaveTime {
			return nil
		}

		time.Sleep(time.Second)
	}

	return errors.New("Timed out waiting for background save to complete")
}

func (client *client) GetConfig(key string) (string, error) {
	configCommand := client.conf.CommandAlias("CONFIG")

	output, err := redisclient.StringMap(client.connection.Do(configCommand, "GET", key))
	if err != nil {
		return "", err
	}

	value, found := output[key]
	if !found {
		return "", fmt.Errorf("Key '%s' not found", key)
	}

	return value, nil
}

func (client *client) setConfig(key string, value string) error {
	configCommand := client.conf.CommandAlias("CONFIG")

	_, err := client.connection.Do(configCommand, "SET", key, value)
	return err
}

func (client *client) info() (map[string]string, error) {
	infoCommand := client.conf.CommandAlias("INFO")

	info := map[string]string{}

	response, err := redisclient.String(client.connection.Do(infoCommand))
	if err != nil {
		return nil, err
	}

	for _, entry := range strings.Split(response, "\n") {
		trimmedEntry := strings.TrimSpace(entry)
		if trimmedEntry == "" || trimmedEntry[0] == '#' {
			continue
		}

		pair := strings.Split(trimmedEntry, ":")
		info[pair[0]] = pair[1]
	}

	return info, nil
}
