package config

import (
	"encoding/json"
	"os"
	"time"

	"github.com/planetsp/k-drive/pkg/logging"
)

type Configuration struct {
	AppName                        string        `json:"appName"`
	WorkingDirectory               string        `json:"workingDirectory"`
	CloudProvider                  string        `json:"cloudProvider"`
	BucketName                     string        `json:"bucketName"`
	LocalDirectoryPollingFrequency time.Duration `json:"localDirectoryPollingFrequency"`
}

var config *Configuration
var configLoaded bool

func init() {
	config = &Configuration{}
	configLoaded = LoadConfig()
}

func LoadConfig() bool {
	file, err := os.Open("conf.json")
	if err != nil {
		logging.Info("Configuration file 'conf.json' not found")
		return false
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
		logging.Error("Failed to parse configuration file: %v", err)
		return false
	}

	logging.Info("Configuration loaded successfully")
	return true
}

func SaveConfig(cfg *Configuration) error {
	config = cfg

	file, err := os.Create("conf.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	err = encoder.Encode(config)
	if err != nil {
		return err
	}

	configLoaded = true
	logging.Info("Configuration saved successfully")
	return nil
}

func GetConfig() *Configuration {
	return config
}

func IsConfigLoaded() bool {
	return configLoaded
}

func CreateDefaultConfig() *Configuration {
	return &Configuration{
		AppName:                        "K-Drive",
		WorkingDirectory:               "",
		CloudProvider:                  "aws s3",
		BucketName:                     "",
		LocalDirectoryPollingFrequency: 3,
	}
}
