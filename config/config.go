package config

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	MQTT_Server struct {
		Host           string `yaml:"host"`
		Port           int    `yaml:"port"`
		Client_name    string `yaml:"client_name"`
		Client_version string `yaml:"client_version"`
	} `yaml:mqtt_server`
	Loging struct {
		Log_file_path string `yaml:"log_file_path"`
		Log_level     string `yaml:"log_level"`
	}
	Devices map[string]struct {
		Device_name   string `yaml:"device_name"`
		Host          string `yaml:"host"`
		Port          string `yaml:"port"`
		Control_topic string `yaml:"control_topic"`
		Satus_topic   string `yaml:"satus_topic"`
	} `yaml:lamps`
}

func NewConfig(path string) (*Config, error) {

	config := &Config{}
	if file, err := os.Open(path); err != nil {
		fmt.Println("Error during opening file. Error:" + err.Error())
		return nil, err
	} else {
		decode := yaml.NewDecoder(file)
		if err := decode.Decode(&config); err != nil {
			return nil, err
		}
		file.Close()
		return config, nil
	}

}

func ValidateConfig(config Config) {
	fmt.Println("Validating configuration")
}
