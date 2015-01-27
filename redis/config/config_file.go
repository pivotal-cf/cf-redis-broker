package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

type configDirectives []configDirective

func (directives configDirectives) String() string {
	lines := bytes.Buffer{}

	for _, directive := range directives {
		lines.WriteString(directive.String() + "\n")
	}

	return lines.String()
}

type configDirective struct {
	keyword   string
	arguments []string
}

func (directive configDirective) String() string {
	return fmt.Sprintf("%s %s", directive.keyword, strings.Join(directive.arguments, " "))
}

func SaveRedisConfAdditions(fromPath string, toPath string, syslogIdentSuffix string) error {
	defaultConfig, err := os.Open(fromPath)
	if err != nil {
		return err
	}
	defer defaultConfig.Close()

	newConfig, err := os.Create(toPath)
	if err != nil {
		return err
	}
	defer newConfig.Close()

	io.Copy(newConfig, defaultConfig)

	// make sure we're starting on a new line
	_, err = newConfig.WriteString("\n")
	if err != nil {
		return err
	}

	redisConf := syslogConfig(syslogIdentSuffix)

	_, err = newConfig.WriteString(redisConf)
	if err != nil {
		return err
	}

	return nil
}

func syslogConfig(syslogIdentSuffix string) string {
	directives := configDirectives{
		configDirective{
			keyword: "syslog-enabled",
			arguments: []string{
				"yes",
			},
		},
		configDirective{
			keyword: "syslog-ident",
			arguments: []string{
				fmt.Sprintf("redis-server-%s", syslogIdentSuffix),
			},
		},
		configDirective{
			keyword: "syslog-facility",
			arguments: []string{
				"local0",
			},
		},
	}

	return directives.String()
}
