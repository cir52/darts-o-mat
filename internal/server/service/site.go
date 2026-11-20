package service

import (
	"context"

	"github.com/sj14/darts-o-mat/internal/server/domain/account"
	"github.com/sj14/darts-o-mat/internal/server/domain/site"
)

// RegistrationOpen says whether anyone may create an account. The setting is read from the
// database every time, so every replica follows what an admin saved.
func (s *Service) RegistrationOpen(ctx context.Context) (bool, error) {
	settings, err := s.store.SiteSettings(ctx)
	return settings.Registration, err
}

// SetRegistration allows or forbids creating accounts. Only an admin may.
func (s *Service) SetRegistration(ctx context.Context, acc account.Account, open bool) error {
	if err := checkAdmin(acc); err != nil {
		return err
	}
	settings, err := s.store.SiteSettings(ctx)
	if err != nil {
		return err
	}
	settings.Registration = open
	return s.store.SetSiteSettings(ctx, settings)
}

// SetCaptcha turns the check on registering that the visitor is a person on or off; it only
// does something if the server has its keys (CaptchaConfigured). Only an admin may.
func (s *Service) SetCaptcha(ctx context.Context, acc account.Account, on bool) error {
	if err := checkAdmin(acc); err != nil {
		return err
	}
	settings, err := s.store.SiteSettings(ctx)
	if err != nil {
		return err
	}
	settings.Captcha = on
	return s.store.SetSiteSettings(ctx, settings)
}

// SiteStats counts what the server holds (accounts, games, darts, boards …) for the admin
// page; ErrForbidden for everyone but an admin.
func (s *Service) SiteStats(ctx context.Context, acc account.Account) (site.Stats, error) {
	if err := checkAdmin(acc); err != nil {
		return site.Stats{}, err
	}
	return s.store.SiteStats(ctx, s.now())
}
