package helpers

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
)

type TemplateData struct {
	DataDir            string
	Host               string
	Port               int
	ConfigDir          string
	LogDir             string
	AwsAccessKey       string
	AwsSecretAccessKey string
	PlanName           string
	BrokerUrl          string
}

func HandleTemplate(sourceFile, destFile string, data interface{}) error {
	configFolderTemplateBytes, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		return err
	}

	templateString := string(configFolderTemplateBytes)

	tmpl, err := template.New(filepath.Base(sourceFile)).Parse(templateString)
	if err != nil {
		return err
	}

	buffer := bytes.NewBufferString("")
	err = tmpl.Execute(buffer, data)
	if err != nil {
		return err
	}

	contents := buffer.String()

	file, err := os.Create(destFile)
	if err != nil {
		return err
	}

	defer file.Close()

	_, err = file.WriteString(contents)

	return err
}
