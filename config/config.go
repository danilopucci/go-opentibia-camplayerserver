package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type World struct {
	Name     string `yaml:"name"`
	ID       int    `yaml:"id"`
	HostName string `yaml:"hostname"`
	Port     uint16 `yaml:"port"`
	HostIP   uint32
}

type LoginServer struct {
	HostName string `yaml:"hostname"`
	Port     int    `yaml:"port"`
}

type CamServer struct {
	HostName string `yaml:"hostname"`
	Port     int    `yaml:"port"`
}

type GameServer struct {
	Worlds []World `yaml:"worlds"`
}

// Config represents the structure of the configuration
type Config struct {
	GameServer   GameServer     `yaml:"gameserver"`
	LoginServer  LoginServer    `yaml:"loginserver"`
	CamServer    CamServer      `yaml:"camserver"`
	Database     DatabaseConfig `yaml:"database"`
	RSAKeyFile   string         `yaml:"rsakeyfile"`
	Motd         string         `yaml:"motd"`
	QueryVersion string         `yaml:"queryversion"`
}

type DatabaseConfig struct {
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	HostName string `yaml:"hostname"`
	Port     int    `yaml:"port"`
}

func LoadConfig() (Config, error) {
	var config Config

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".") // Set the path to look for the config file in the current directory

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return config, fmt.Errorf("error reading config file: %w", err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		return config, fmt.Errorf("unable to decode into struct: %w", err)
	}

	//convertConfigWorldHostnameToIp(&config)

	return config, nil
}

func GetWorldById(config Config, worldId int) (World, error) {
	var world World

	for _, w := range config.GameServer.Worlds {
		if w.ID == worldId {
			world = w
			return world, nil
		}
	}

	return world, fmt.Errorf("could not find any world with id %d", worldId)
}

func GetDefaultWorld(config *Config) World {
	return config.GameServer.Worlds[0]
}

// func convertConfigWorldHostnameToIp(config *Config) {
// 	for i := range config.GameServer.Worlds {
// 		ipAddress, err := utils.IpToUint32(config.GameServer.Worlds[i].HostName)
// 		if err == nil {
// 			config.GameServer.Worlds[i].HostIP = ipAddress
// 		} else {
// 			fmt.Printf("could not convert world %s host %s to number ip address: %s\n", config.GameServer.Worlds[i].Name, config.GameServer.Worlds[i].HostName, err)
// 		}
// 	}
// }
