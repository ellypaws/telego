package lib

import "github.com/joho/godotenv"

func Set(key string, value string) error {
	envs, err := godotenv.Read()
	if err != nil {
		return err
	}

	envs[key] = value

	return godotenv.Write(envs, ".env")
}
