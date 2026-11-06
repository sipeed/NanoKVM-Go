package tailscale

import (
	"errors"

	"NanoKVM-Server/config"
)

var (
	errSipeedDisclaimerRequired   = errors.New("sipeed disclaimer acceptance is required")
	errSipeedDisclaimerSaveFailed = errors.New("save sipeed disclaimer acceptance failed")
)

type sipeedDisclaimerStore interface {
	Accepted() bool
	SetAccepted(bool) error
}

type configSipeedDisclaimerStore struct{}

func (configSipeedDisclaimerStore) Accepted() bool {
	return config.GetSipeedDisclaimerAccepted()
}

func (configSipeedDisclaimerStore) SetAccepted(accepted bool) error {
	return config.SetSipeedDisclaimerAccepted(accepted)
}

func (s *Service) sipeedDisclaimerAccepted() bool {
	return s != nil && s.disclaimerStore != nil && s.disclaimerStore.Accepted()
}

func (s *Service) setSipeedDisclaimerAccepted(accepted bool) error {
	if s.disclaimerStore == nil || s.disclaimerStore.SetAccepted(accepted) != nil {
		return errSipeedDisclaimerSaveFailed
	}
	return nil
}

// SipeedDisclaimerAccepted reports whether the device owner already accepted the
// Sipeed privacy terms. It is the exported read surface for transports that
// gate their own flow on persisted consent (the root-local Tailscale gate).
func (s *Service) SipeedDisclaimerAccepted() bool {
	return s.sipeedDisclaimerAccepted()
}

// GrantSipeedDisclaimer idempotently persists the privacy-terms acceptance. It
// is the exported write surface for transports that must record consent before
// any upstream contact. A nil service maps to the save-failure sentinel rather
// than panicking, so the gate wiring can answer a stable failure.
func (s *Service) GrantSipeedDisclaimer() error {
	if s == nil {
		return errSipeedDisclaimerSaveFailed
	}
	return s.setSipeedDisclaimerAccepted(true)
}
