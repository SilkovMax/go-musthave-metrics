package config

import (
	"flag"
	"os"
	"strconv"
)

// IsFlagSet проверяет, был ли флаг передан явно в командной строке.
func IsFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// ResolveString возвращает значение по приоритету: Флаг > Env > JSON > Default.
func ResolveString(flagVal, envName string, jsonVal *string, defaultVal string, flagName string) string {
	if IsFlagSet(flagName) {
		return flagVal
	}
	if envVal, ok := os.LookupEnv(envName); ok {
		return envVal
	}
	if jsonVal != nil {
		return *jsonVal
	}
	return defaultVal
}

// ResolveInt возвращает значение по приоритету: Флаг > Env > JSON > Default.
func ResolveInt(flagVal int, envName string, jsonVal *int, defaultVal int, flagName string) int {
	if IsFlagSet(flagName) {
		return flagVal
	}
	if envVal, ok := os.LookupEnv(envName); ok {
		if v, err := strconv.Atoi(envVal); err == nil {
			return v
		}
	}
	if jsonVal != nil {
		return *jsonVal
	}
	return defaultVal
}

// ResolveBool возвращает значение по приоритету: Флаг > Env > JSON > Default.
func ResolveBool(flagVal bool, envName string, jsonVal *bool, defaultVal bool, flagName string) bool {
	if IsFlagSet(flagName) {
		return flagVal
	}
	if envVal, ok := os.LookupEnv(envName); ok {
		if v, err := strconv.ParseBool(envVal); err == nil {
			return v
		}
	}
	if jsonVal != nil {
		return *jsonVal
	}
	return defaultVal
}
