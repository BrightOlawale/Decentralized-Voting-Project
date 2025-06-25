package config

import (
    "github.com/spf13/viper"
)

type Config struct {
    Server struct {
        Port string `mapstructure:"port"`
        Host string `mapstructure:"host"`
    } `mapstructure:"server"`
    
    Database struct {
        Type     string `mapstructure:"type"`
        Path     string `mapstructure:"path"`
        Host     string `mapstructure:"host"`
        Port     int    `mapstructure:"port"`
        User     string `mapstructure:"user"`
        Password string `mapstructure:"password"`
        DBName   string `mapstructure:"dbname"`
    } `mapstructure:"database"`
    
    Blockchain struct {
        NetworkURL      string `mapstructure:"network_url"`
        ContractAddress string `mapstructure:"contract_address"`
        PrivateKey      string `mapstructure:"private_key"`
    } `mapstructure:"blockchain"`
    
    Biometric struct {
        FingerprintDevice string  `mapstructure:"fingerprint_device"`
        QualityThreshold  float64 `mapstructure:"quality_threshold"`
        MatchThreshold    float64 `mapstructure:"match_threshold"`
    } `mapstructure:"biometric"`
}

func LoadConfig(path string) (*Config, error) {
    viper.SetConfigFile(path)
    viper.AutomaticEnv()
    
    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }
    
    return &config, nil
}
