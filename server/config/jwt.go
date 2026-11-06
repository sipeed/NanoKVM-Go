package config

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/spf13/viper"
)

var readJWTSecretRandom = rand.Read

// RegenerateSecretKey regenerate secret key when logout
func RegenerateSecretKey() error {
	conf := GetInstance()
	if conf.JWT.RevokeTokensOnLogout {
		return RegenerateJWTSecretKey()
	}
	return nil
}

func RegenerateJWTSecretKey() error {
	configMu.Lock()
	defer configMu.Unlock()
	key, err := generateRandomSecretKey()
	if err != nil {
		return err
	}

	updated := instance
	updated.JWT.SecretKey = key
	if err := save(updated); err != nil {
		return err
	}

	instance.JWT.SecretKey = key
	viper.Set("jwt.secretKey", key)
	return nil
}

// Generate random string for secret key.
func generateRandomSecretKey() (string, error) {
	b := make([]byte, 64)
	if _, err := readJWTSecretRandom(b); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(b), nil
}
