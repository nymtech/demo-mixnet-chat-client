// various types related to chat client. Mostly to deal with circular dependencies
package types

import "github.com/nymtech/nym-mixnet/config"

type Session struct {
	recipient      config.ClientConfig
	recipientAlias string
	sessionNonce   int64
}

func (s *Session) Recipient() config.ClientConfig {
	return s.recipient
}

func (s *Session) UpdateAlias(alias string) {
	s.recipientAlias = alias
}

func (s *Session) RecipientAlias() string {
	return s.recipientAlias
}

func (s *Session) IncrementNonce() int64 {
	s.sessionNonce++
	return s.sessionNonce
}

func NewSession(recipient config.ClientConfig, alias string) *Session {
	return &Session{
		recipient:      recipient,
		recipientAlias: alias,
		sessionNonce:   0,
	}
}
