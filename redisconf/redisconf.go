package redisconf

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
)

type Param struct {
	Key   string
	Value string
}

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

func (conf Conf) Password() string {
	pass := []byte(conf.Get("requirepass"))
	if len(pass) > 0 && pass[0] == '"' && pass[len(pass)-1] == '"' {
		pass = pass[1 : len(pass)-1]
	}
	return string(pass)
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

func (conf *Conf) CommandAlias(command string) string {
	alias, ok := conf.commandMapping()[command]
	if !ok {
		return command
	}
	return alias
}

func (conf *Conf) commandMapping() map[string]string {
	renamedCommands := conf.getAll("rename-command")
	commandMapping := make(map[string]string)
	for _, param := range renamedCommands {
		args := strings.Split(param.Value, " ")
		original := args[0]
		alias := strings.Replace(args[1], "\"", "", -1)
		commandMapping[original] = alias
	}
	return commandMapping
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

func CopyWithInstanceAdditions(fromPath, toPath, syslogIdentSuffix, port, password string) error {
	defaultConfig, err := Load(fromPath)
	if err != nil {
		return err
	}

	defaultConfig.Set("syslog-enabled", "yes")
	defaultConfig.Set("syslog-ident", fmt.Sprintf("redis-server-%s", syslogIdentSuffix))
	defaultConfig.Set("syslog-facility", "local0")

	defaultConfig.Set("port", port)
	defaultConfig.Set("requirepass", password)

	err = defaultConfig.Save(toPath)
	if err != nil {
		return err
	}

	return nil
}
