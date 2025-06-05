package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	_ "embed"
)

//go:embed config.toml.tpl
var defaultConfigTemplate string

var configTemplate *template.Template

func init() {
	var err error
	tmpl := template.New("configFileTemplate").Funcs(template.FuncMap{
		"StringsJoin": strings.Join,
	})
	if configTemplate, err = tmpl.Parse(defaultConfigTemplate); err != nil {
		panic(err)
	}
}

func WriteConfigFile(configFilePath string, config *Config) error {
	var buffer bytes.Buffer

	if err := configTemplate.Execute(&buffer, config); err != nil {
		panic(err)
	}

	MustWriteFile(configFilePath, buffer.Bytes(), 0o644)

	return nil
}

func WriteDefaultConfigFile(configFilePath string) error {
	return WriteConfigFile(configFilePath, DefaultConfig())
}

func WriteFile(filePath string, contents []byte, mode os.FileMode) error {
	return os.WriteFile(filePath, contents, mode)
}

func MustWriteFile(filePath string, contents []byte, mode os.FileMode) {
	err := WriteFile(filePath, contents, mode)
	if err != nil {
		Exit(fmt.Sprintf("MustWriteFile failed: %v", err))
	}
}

func Exit(s string) {
	fmt.Println(s)
	os.Exit(1)
}
