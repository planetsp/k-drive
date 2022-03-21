package config

import (
	"encoding/json"
	"os"
	"time"

	"github.com/planetsp/k-drive/pkg/logging"
)

type Configuration struct {
	AppName                        string
	WorkingDirectory               string
	CloudProvider                  string
	BucketName                     string
	LocalDirectoryPollingFrequency time.Duration
}

var config *Configuration

func init() {
	config = &Configuration{}
	file, _ := os.Open("conf.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	config = &Configuration{}
	err := decoder.Decode(config)
	if err != nil {
		logging.Error(err)
	}
}
func GetConfig() *Configuration {
	return config
}
