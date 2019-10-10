// various types related to chat client. Mostly to deal with circular dependencies
package types

import "github.com/nymtech/nym-mixnet/config"

type Session struct {
	recipient    config.ClientConfig
	sessionNonce int64
}

func (s *Session) Recipient() config.ClientConfig {
	return s.recipient
}

func (s *Session) IncrementNonce() int64 {
	s.sessionNonce++
	return s.sessionNonce
}

func NewSession(recipient config.ClientConfig) *Session {
	return &Session{
		recipient:    recipient,
		sessionNonce: 0,
	}
}

