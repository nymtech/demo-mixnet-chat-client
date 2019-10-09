package alias

import (
	"errors"
	"github.com/nymtech/demo-mixnet-chat-client/storage"
	"github.com/nymtech/nym-mixnet/sphinx"
)

type Alias struct {
	AssignedName string
	PublicKey *sphinx.PublicKey
	ProviderPublicKey *sphinx.PublicKey
}


type AliasStore interface {
	StoreAlias(alias *Alias)
	GetAlias(*sphinx.PublicKey, *sphinx.PublicKey) *Alias
	RemoveAlias(alias *Alias)
	GetAllAliases() []*Alias
	RemoveAllAliases()
}


func NewAliasStore(typ string, fileName, fileDir *string) (AliasStore, error) {
	switch typ {
	case "leveldb", "goleveldb":
		return storage.New(*fileName, *fileDir)
	default:
		return nil, errors.New("unsupported alias store type")
	}
}
