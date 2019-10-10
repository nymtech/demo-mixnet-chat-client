package alias

import (
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
