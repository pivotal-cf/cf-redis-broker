package client

import (
	"errors"
	"fmt"
	"strings"
	"time"

	redisclient "github.com/garyburd/redigo/redis"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

func Connect(host string, conf redisconf.Conf) (*Client, error) {
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

	return &Client{conf: conf, connection: connection}, err
}

type Client struct {
	conf       redisconf.Conf
	connection redisclient.Conn
}

func (client *Client) WaitUntilRedisNotLoading(timeoutMilliseconds int) error {
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

func (client *Client) CreateSnapshot(timeoutInSeconds int) error {
	lastSaveTime, err := client.LastRDBSaveTime()
	if err != nil {
		return err
	}

	client.waitForUniqueSnapshotTime()

	err = client.runBGSave()
	if err != nil {
		return err
	}

	err = client.waitForNewSaveSince(lastSaveTime, timeoutInSeconds)
	if err != nil {
		return err
	}

	return nil
}

func (client *Client) EnableAOF() error {
	return client.setConfig("appendonly", "yes")
}

func (client *Client) runBGSave() error {
	bgSaveCommand := client.conf.CommandAlias("BGSAVE")
	_, err := client.connection.Do(bgSaveCommand)
	return err
}

func (client *Client) LastRDBSaveTime() (int64, error) {
	saveTimeStr, err := client.connection.Do("LASTSAVE")
	if err != nil {
		return 0, err
	}

	return saveTimeStr.(int64), nil
}

func (client *Client) InfoField(fieldName string) (string, error) {
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

func (client *Client) waitForUniqueSnapshotTime() {
	time.Sleep(time.Second)
}

func (client *Client) waitForNewSaveSince(lastSaveTime int64, timeoutInSeconds int) error {
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

func (client *Client) setConfig(key string, value string) error {
	configCommand := client.conf.CommandAlias("CONFIG")

	_, err := client.connection.Do(configCommand, "SET", key, value)
	return err
}

func (client *Client) info() (map[string]string, error) {
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
