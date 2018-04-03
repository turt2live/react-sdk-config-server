package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"gopkg.in/yaml.v2"
)

type runtimeConfig struct {
	MigrationsPath string
}

var Runtime = &runtimeConfig{}

type GeneralConfig struct {
	BindAddress  string `yaml:"bindAddress"`
	Port         int    `yaml:"port"`
	LogDirectory string `yaml:"logDirectory"`
}

type DatabaseConfig struct {
	Postgres string `yaml:"postgres"`
}

type ApiConfig struct {
	SharedSecret string `yaml:"sharedSecret"`
}

type ConfigServerConfig struct {
	General   *GeneralConfig  `yaml:"repo"`
	Database  *DatabaseConfig `yaml:"database"`
	ApiConfig *ApiConfig      `yaml:"api""`
}

var instance *ConfigServerConfig
var singletonLock = &sync.Once{}
var Path = "config-server.yaml"

func ReloadConfig() (error) {
	c := NewDefaultConfig()

	// Write a default config if the one given doesn't exist
	_, err := os.Stat(Path)
	exists := err == nil || !os.IsNotExist(err)
	if !exists {
		fmt.Println("Generating new configuration...")
		configBytes, err := yaml.Marshal(c)
		if err != nil {
			return err
		}

		newFile, err := os.Create(Path)
		if err != nil {
			return err
		}

		_, err = newFile.Write(configBytes)
		if err != nil {
			return err
		}

		err = newFile.Close()
		if err != nil {
			return err
		}
	}

	f, err := os.Open(Path)
	if err != nil {
		return err
	}
	defer f.Close()

	buffer, err := ioutil.ReadAll(f)
	err = yaml.Unmarshal(buffer, &c)
	if err != nil {
		return err
	}

	instance = c
	return nil
}

func Get() (*ConfigServerConfig) {
	if instance == nil {
		singletonLock.Do(func() {
			err := ReloadConfig()
			if err != nil {
				panic(err)
			}
		})
	}
	return instance
}

const DefaultSharedSecret = "CHANGE_ME"
func NewDefaultConfig() *ConfigServerConfig {
	return &ConfigServerConfig{
		General: &GeneralConfig{
			BindAddress:  "127.0.0.1",
			Port:         8000,
			LogDirectory: "logs",
		},
		Database: &DatabaseConfig{
			Postgres: "postgres://your_username:your_password@localhost/database_name?sslmode=disable",
		},
		ApiConfig: &ApiConfig{
			SharedSecret: DefaultSharedSecret,
		},
	}
}
