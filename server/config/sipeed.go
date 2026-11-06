package config

import "github.com/spf13/viper"

func GetSipeedDisclaimerAccepted() bool {
	GetInstance()
	configMu.Lock()
	defer configMu.Unlock()
	return instance.SipeedRemoteNetwork.DisclaimerAccepted
}

func SetSipeedDisclaimerAccepted(accepted bool) error {
	GetInstance()
	configMu.Lock()
	defer configMu.Unlock()

	updated := instance
	updated.SipeedRemoteNetwork.DisclaimerAccepted = accepted
	if err := save(updated); err != nil {
		return err
	}
	instance.SipeedRemoteNetwork.DisclaimerAccepted = accepted
	viper.Set("sipeedRemoteNetwork.disclaimerAccepted", accepted)
	return nil
}
