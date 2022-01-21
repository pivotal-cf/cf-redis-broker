package redisconf

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pborman/uuid"

	"os"

	"github.com/cloudfoundry/gosigar"
)

type Param struct {
	Key   string
	Value string
}

const (
	DefaultHost = "127.0.0.1"
	DefaultPort = 6379
	DefaultTLSPort = 16379
)

type Conf []Param

func New(params ...Param) Conf {
	return Conf(params)
}

func Load(path string) (Conf, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return decode(data)
}

func (conf Conf) Save(path string) error {
	data := conf.Encode()
	return ioutil.WriteFile(path, data, 0644)
}

func decode(data []byte) (Conf, error) {
	conf := []Param{}

	scanner := bufio.NewScanner(bytes.NewBuffer(data))

	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		switch line[:1] {
		case "#":
			continue
		case " ":
			continue
		case "\t":
			continue
		case "\n":
			continue
		}

		param, err := parseParam(line)
		if err != nil {
			return nil, err
		}

		conf = append(conf, param)
	}

	return conf, nil
}

func (conf Conf) Host() string {
	host := conf.Get("bind")
	if host == "" {
		return DefaultHost
	}
	return host
}

func (conf Conf) Port() int {
	port, err := strconv.Atoi(conf.Get("port"))
	if err != nil {
		return DefaultPort
	}
	if port == 0 {
		port, err = strconv.Atoi(conf.Get("tls-port"))
		if err != nil {
			return DefaultTLSPort
		}
	}
	return port
}

func (conf Conf) Password() string {
	return conf.Get("requirepass")
}

func (conf Conf) Get(key string) string {
	params := conf.getAll(key)
	if len(params) < 1 {
		return ""
	}
	return params[0].Value
}

func (conf Conf) HasKey(key string) bool {
	for _, param := range conf {
		if key == param.Key {
			return true
		}
	}
	return false
}

func (conf Conf) getAll(key string) []Param {
	params := []Param{}
	for _, param := range conf {
		if key == param.Key {
			params = append(params, param)
		}
	}
	return params
}

func (conf *Conf) CommandAliases() map[string]string {
	renamedCommands := conf.getAll("rename-command")
	commandAliases := make(map[string]string)
	for _, param := range renamedCommands {
		args := strings.Split(param.Value, " ")
		original := args[0]
		alias := strings.Replace(args[1], "\"", "", -1)
		commandAliases[original] = alias
	}
	return commandAliases
}

func (conf *Conf) Set(key string, value string) {
	newParam := Param{
		Key:   key,
		Value: value,
	}

	// update
	for index, param := range *conf {
		if key == param.Key {
			(*conf)[index] = newParam
			return
		}
	}

	// insert
	*conf = append(*conf, newParam)
}

func (conf Conf) Encode() []byte {
	output := []byte{}

	for _, param := range conf {
		line := param.Key + " " + param.Value + "\n"
		output = append(output, []byte(line)...)
	}

	return output
}

func parseParam(line string) (Param, error) {
	words := strings.SplitN(line, " ", 2)
	if len(words) != 2 {
		msg := fmt.Sprintf("Unable to split redis.conf parameter into key/value pair: %s", line)
		return Param{}, errors.New(msg)
	}

	return Param{
		Key:   words[0],
		Value: words[1],
	}, nil
}

func CopyWithInstanceAdditions(fromPath, toPath, instanceID, port, password, pidDir string) error {
	defaultConfig, err := Load(fromPath)
	if err != nil {
		return err
	}

	defaultConfig.Set("syslog-enabled", "yes")
	defaultConfig.Set("syslog-ident", fmt.Sprintf("redis-server-%s", instanceID))
	defaultConfig.Set("syslog-facility", "local0")

	defaultConfig.Set("port", port)
	defaultConfig.Set("requirepass", password)

	defaultConfig.Set("pidfile", filepath.Join(pidDir, instanceID+".pid"))

	err = defaultConfig.Save(toPath)
	os.Chmod(toPath, 0640)
	if err != nil {
		return err
	}

	return nil
}

func calculateMaxMemory() (int, error) {
	mem := sigar.Mem{}
	if err := mem.Get(); err != nil {
		return 0, err
	}

	return int(float64(mem.Total) * 0.45), nil
}

func (c *Conf) setMaxMemory() error {
	maxMem, err := calculateMaxMemory()
	if err != nil {
		return err
	}
	c.Set("maxmemory", strconv.Itoa(maxMem))
	return nil
}

func (c *Conf) setRandomPassword() {
	c.setPassword(uuid.NewRandom().String())
}

func (c *Conf) setPassword(password string) {
	c.Set("requirepass", password)
}
