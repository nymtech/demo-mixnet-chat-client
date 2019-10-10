package storage

import "github.com/nymtech/demo-mixnet-chat-client/chat-client/commands/alias"

// requirements for any store for the chat
type ChatStore interface {
	alias.AliasStore
}
