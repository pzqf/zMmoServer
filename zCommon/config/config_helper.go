package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pzqf/zUtil/zConfig"
)

func GetConfigString(cfg *zConfig.Config, key string, defaultValue string) string {
	if value, err := cfg.GetString(key); err == nil {
		return value
	}
	return defaultValue
}

func GetConfigInt(cfg *zConfig.Config, key string, defaultValue int) int {
	if value, err := cfg.GetInt(key); err == nil {
		return value
	}
	return defaultValue
}

func GetConfigBool(cfg *zConfig.Config, key string, defaultValue bool) bool {
	if value, err := cfg.GetBool(key); err == nil {
		return value
	}
	return defaultValue
}

func GetConfigIntSlice(cfg *zConfig.Config, key string, defaultValue []int) []int {
	if value, err := cfg.GetString(key); err == nil {
		strs := strings.Split(value, ",")
		ints := make([]int, 0, len(strs))
		for _, str := range strs {
			str = strings.TrimSpace(str)
			if str != "" {
				if i, err := strconv.Atoi(str); err == nil {
					ints = append(ints, i)
				}
			}
		}
		if len(ints) > 0 {
			return ints
		}
	}
	return defaultValue
}

func GetConfigFloat(cfg *zConfig.Config, key string, defaultValue float64) float64 {
	if value, err := cfg.GetFloat(key); err == nil {
		return value
	}
	return defaultValue
}

func GetEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func GetEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func GetEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func ReplacePlaceholder(s, placeholder string, value int) string {
	return strings.Replace(s, placeholder, fmt.Sprintf("%d", value), -1)
}
