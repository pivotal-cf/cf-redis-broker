package client

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	redisclient "github.com/gomodule/redigo/redis"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

//go:generate counterfeiter -o fakes/fake_client.go . Client
type Client interface {
	Disconnect() error
	WaitUntilRedisNotLoading(timeoutMilliseconds int) error
	EnableAOF() error
	LastRDBSaveTime() (int64, error)
	Info() (map[string]string, error)
	InfoField(fieldName string) (string, error)
	GlobalKeyCount() (int, error)
	GetConfig(key string) (string, error)
	RDBPath() (string, error)
	Address() string
	WaitForNewSaveSince(lastSaveTime int64, timeout time.Duration) error
	RunBGSave() error
	Ping() error
	Exec(command string, args ...interface{}) (interface{}, error)
}

type client struct {
	host     string
	port     int
	password string
	tls      bool
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

func Tls(tls bool) Option {
	return func(c *client) {
		c.tls = tls
	}
}

func Connect(options ...Option) (Client, error) {
	c := &client{
		host:    "0.0.0.0",
		port:    redisconf.DefaultPort,
		aliases: map[string]string{},
		tls:     false,
	}

	for _, opt := range options {
		opt(c)
	}

	address := fmt.Sprintf("%v:%v", c.host, c.port)

	var err error
	c.connection, err = redisclient.Dial("tcp", address, redisclient.DialUseTLS(c.tls), redisclient.DialTLSSkipVerify(c.tls))
	if err != nil {
		return nil, err
	}

	if c.password != "" {
		if _, err := c.Exec("AUTH", c.password); err != nil {
			c.connection.Close()
			return nil, err
		}
	}

	return c, nil
}

func (c *client) Exec(command string, args ...interface{}) (interface{}, error) {
	return c.connection.Do(command, args...)
}

func (c *client) Disconnect() error {
	return c.connection.Close()
}

func (c *client) Address() string {
	return fmt.Sprintf("%s:%d", c.host, c.port)
}

func (c *client) WaitUntilRedisNotLoading(timeoutMilliseconds int) error {
	for i := 0; i < timeoutMilliseconds; i += 100 {
		loading, err := c.InfoField("loading")
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

func (c *client) EnableAOF() error {
	return c.setConfig("appendonly", "yes")
}

func (c *client) RunBGSave() error {
	_, err := c.Exec(c.lookupAlias("BGSAVE"))
	return err
}

func (c *client) LastRDBSaveTime() (int64, error) {
	saveTimeStr, err := c.Exec("LASTSAVE")
	if err != nil {
		return 0, err
	}

	return saveTimeStr.(int64), nil
}

func (c *client) InfoField(fieldName string) (string, error) {
	info, err := c.Info()
	if err != nil {
		return "", fmt.Errorf("Error during redis info: %s", err.Error())
	}

	value, ok := info[fieldName]
	if !ok {
		return "", fmt.Errorf("Unknown field: %s", fieldName)
	}

	return value, nil
}

func (c *client) Info() (map[string]string, error) {
	infoCommand := c.lookupAlias("INFO")

	info := map[string]string{}

	response, err := redisclient.String(c.Exec(infoCommand))
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

func (c *client) GlobalKeyCount() (int, error) {
	infoCommand := c.lookupAlias("INFO")

	keyspaces, err := redisclient.String(c.Exec(infoCommand, "keyspace"))
	if err != nil {
		return 0, err
	}

	globalCount := 0
	regex := regexp.MustCompile(`keys=(\d+)`)

	for _, keyspace := range strings.Split(keyspaces, "\n") {
		matches := regex.FindStringSubmatch(keyspace)
		if len(matches) != 2 {
			continue
		}

		count, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		globalCount += count
	}

	return globalCount, nil
}

func (c *client) WaitForNewSaveSince(lastSaveTime int64, timeout time.Duration) error {
	timer := time.After(timeout)
	for {
		select {
		case <-time.After(time.Second):
			latestSaveTime, err := c.LastRDBSaveTime()
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

func (c *client) GetConfig(key string) (string, error) {
	configCommand := c.lookupAlias("CONFIG")

	output, err := redisclient.StringMap(c.Exec(configCommand, "GET", key))
	if err != nil {
		return "", err
	}

	value, found := output[key]
	if !found {
		return "", fmt.Errorf("Key '%s' not found", key)
	}

	return value, nil
}

func (c *client) RDBPath() (string, error) {
	dataDir, err := c.GetConfig("dir")
	if err != nil {
		return "", err
	}

	if dataDir == "" {
		return "", errors.New("Data dir not set")
	}

	dbFilename, err := c.GetConfig("dbfilename")
	if err != nil {
		return "", err
	}

	return filepath.Join(dataDir, dbFilename), nil
}

func (c *client) setConfig(key string, value string) error {
	configCommand := c.lookupAlias("CONFIG")

	_, err := c.Exec(configCommand, "SET", key, value)
	return err
}

func (c *client) Ping() error {
	pingCommand := c.lookupAlias("PING")

	response, err := redisclient.String(c.Exec(pingCommand))
	if err != nil {
		return err
	}

	if strings.TrimSpace(response) != "PONG" {
		return fmt.Errorf("ping resonded with `%s`", response)
	}

	return nil
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

// Used by redis-backups
func CmdAliases(aliases map[string]string) Option {
	return func(c *client) {
		c.aliases = map[string]string{}
		for cmd, alias := range aliases {
			c.registerAlias(cmd, alias)
		}
	}
}
