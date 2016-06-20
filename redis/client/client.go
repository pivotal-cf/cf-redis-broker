package client

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	redisclient "github.com/garyburd/redigo/redis"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

type client struct {
	host     string
	port     int
	password string
	aliases  map[string]string

	connection redisclient.Conn
}

type Option func(*client)

func Password(password string) Option {
	return func(c *client) {
		c.password = password
	}
}

func Port(port int) Option {
	return func(c *client) {
		if port > 0 {
			c.port = port
		}
	}
}

func Host(host string) Option {
	return func(c *client) {
		c.host = host
	}
}

func CmdAliases(aliases map[string]string) Option {
	return func(c *client) {
		c.aliases = map[string]string{}
		for cmd, alias := range aliases {
			c.registerAlias(cmd, alias)
		}
	}
}

func (c *client) registerAlias(cmd, alias string) {
	c.aliases[strings.ToUpper(cmd)] = alias
}

func (c *client) lookupAlias(cmd string) string {
	alias, found := c.aliases[strings.ToUpper(cmd)]
	if !found {
		return cmd
	}
	return alias
}

func Connect(options ...Option) (Client, error) {
	client := &client{
		host:    "127.0.0.1",
		port:    redisconf.DefaultPort,
		aliases: map[string]string{},
	}

	for _, opt := range options {
		opt(client)
	}

	address := fmt.Sprintf("%v:%v", client.host, client.port)

	var err error
	client.connection, err = redisclient.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	if client.password != "" {
		if _, err := client.connection.Do("AUTH", client.password); err != nil {
			client.connection.Close()
			return nil, err
		}
	}

	return client, nil
}

type Client interface {
	Disconnect() error
	WaitUntilRedisNotLoading(timeoutMilliseconds int) error
	EnableAOF() error
	LastRDBSaveTime() (int64, error)
	Info() (map[string]string, error)
	InfoField(fieldName string) (string, error)
	GetConfig(key string) (string, error)
	RDBPath() (string, error)
	Address() string
	WaitForNewSaveSince(lastSaveTime int64, timeout time.Duration) error
	RunBGSave() error
	Ping() error
}

func (client *client) Disconnect() error {
	return client.connection.Close()
}

func (client *client) Address() string {
	return fmt.Sprintf("%s:%d", client.host, client.port)
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

func (client *client) EnableAOF() error {
	return client.setConfig("appendonly", "yes")
}

func (client *client) RunBGSave() error {
	_, err := client.connection.Do(client.lookupAlias("BGSAVE"))
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
	info, err := client.Info()
	if err != nil {
		return "", fmt.Errorf("Error during redis info: %s" + err.Error())
	}

	value, ok := info[fieldName]
	if !ok {
		return "", errors.New(fmt.Sprintf("Unknown field: %s", fieldName))
	}

	return value, nil
}

func (client *client) WaitForNewSaveSince(lastSaveTime int64, timeout time.Duration) error {
	timer := time.After(timeout)
	for {
		select {
		case <-time.After(time.Second):
			latestSaveTime, err := client.LastRDBSaveTime()
			if err != nil {
				return err
			}

			if latestSaveTime > lastSaveTime {
				return nil
			}
		case <-timer:
			return errors.New("Timed out waiting for background save to complete")
		}
	}
}

func (client *client) GetConfig(key string) (string, error) {
	configCommand := client.lookupAlias("CONFIG")

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

func (client *client) RDBPath() (string, error) {
	dataDir, err := client.GetConfig("dir")
	if err != nil {
		return "", err
	}

	if dataDir == "" {
		return "", errors.New("Data dir not set")
	}

	dbFilename, err := client.GetConfig("dbfilename")
	if err != nil {
		return "", err
	}

	return filepath.Join(dataDir, dbFilename), nil
}

func (client *client) setConfig(key string, value string) error {
	configCommand := client.lookupAlias("CONFIG")

	_, err := client.connection.Do(configCommand, "SET", key, value)
	return err
}

func (client *client) Info() (map[string]string, error) {
	infoCommand := client.lookupAlias("INFO")

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

func (client *client) Ping() error {
	pingCommand := client.lookupAlias("PING")

	response, err := redisclient.String(client.connection.Do(pingCommand))
	if err != nil {
		return err
	}

	if strings.TrimSpace(response) != "PONG" {
		return fmt.Errorf("ping resonded with `%s`", response)
	}

	return nil
}
