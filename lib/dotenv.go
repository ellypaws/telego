package lib

import (
	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

func Set(key string, value string) error {
	envs, err := godotenv.Read()
	if err != nil {
		return err
	}

	envs[key] = value

	return godotenv.Write(envs, ".env")
}

func SetAll(envs map[string]string) error {
	if len(envs) == 0 {
		return nil
	}

	exist, err := godotenv.Read()
	if err != nil {
		return err
	}

	for key, value := range envs {
		exist[key] = value
	}

	return godotenv.Write(exist, ".env")
}

func SetWithLog(logger *log.Logger, envs map[string]string) error {
	if logger == nil {
		return SetAll(envs)
	}

	if len(envs) == 0 {
		logger.Info("No environment variables to set")
		return nil
	}

	exist, err := godotenv.Read()
	if err != nil {
		logger.Error("Error reading .env file:", "err", err)
		return err
	}

	for key, value := range envs {
		logger.Info("Setting environment variable", "key", key, "value", value)
		exist[key] = value
	}

	if err := godotenv.Write(exist, ".env"); err != nil {
		logger.Error("Error writing .env file:", "err", err)
		return err
	}

	return nil
}
