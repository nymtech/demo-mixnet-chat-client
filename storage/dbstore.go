// storage for everything required by the chat-client. From persistent local state to client aliases
package storage

import (
	"github.com/nymtech/demo-mixnet-chat-client/chat-client/commands/alias"
	"github.com/nymtech/nym-mixnet/sphinx"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/util"
	"path/filepath"
)

var (
	aliasPrefix = []byte("ALIAS")
)

// DbStore represents all data required to interact with the storage.
type DbStore struct {
	db *leveldb.DB
}

// get gets the value corresponding to particular key. Returns nil if it doesn't exist.
func (db *DbStore) get(key []byte) []byte {
	key = nonNilBytes(key)
	res, err := db.db.Get(key, nil)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil
		}
		panic(err)
	}
	return res
}

// set sets particular key value pair.
func (db *DbStore) set(key []byte, value []byte) {
	key = nonNilBytes(key)
	value = nonNilBytes(value)
	if err := db.db.Put(key, value, nil); err != nil {
		panic(err)
	}
}

// delete removes particular key value pair.
func (db *DbStore) delete(key []byte) {
	key = nonNilBytes(key)
	if err := db.db.Delete(key, nil); err != nil {
		panic(err)
	}
}

// --------- ALIAS RELATED -----------

// Each alias corresponds to the tuple of user's public key and the public key of it's provider
// each entry follows the structure of: [ ALIAS_PREFIX || PUBLIC_KEY || PROVIDER_PUBLIC_KEY ] -- ALIAS

func (db *DbStore) makeAliasKeyEntry(targetPub, providerPub *sphinx.PublicKey) []byte {
	if targetPub == nil || providerPub == nil {
		return []byte{}
	}
	key := make([]byte, len(aliasPrefix)+2*sphinx.PublicKeySize)
	i := copy(key, aliasPrefix)
	i += copy(key[i:], targetPub.Bytes())
	copy(key[i:], providerPub.Bytes())
	return key
}

func (db *DbStore) recoverKeysFromAliasKeyField(key []byte) (*sphinx.PublicKey, *sphinx.PublicKey) {
	if len(key) != len(aliasPrefix)+2*sphinx.PublicKeySize {
		return nil, nil
	}
	targetPub := new(sphinx.PublicKey)
	providerPub := new(sphinx.PublicKey)

	i := len(aliasPrefix)
	if targetPub.UnmarshalBinary(key[i:i+sphinx.PublicKeySize]) != nil {
		return nil, nil
	}
	i += sphinx.PublicKeySize
	if providerPub.UnmarshalBinary(key[i:i+sphinx.PublicKeySize]) != nil {
		return nil, nil
	}

	return targetPub, providerPub
}

func (db *DbStore) StoreAlias(alias *alias.Alias) {
	key := db.makeAliasKeyEntry(alias.PublicKey, alias.ProviderPublicKey)
	// even if the entry already exists, overwrite it
	db.set(key, []byte(alias.AssignedName))
}

func (db *DbStore) GetAlias(targetPub, providerPub *sphinx.PublicKey) *alias.Alias {
	key := db.makeAliasKeyEntry(targetPub, providerPub)
	aliasB := db.get(key)
	assignedName := ""
	if aliasB != nil {
		assignedName = string(aliasB)
	}

	return &alias.Alias{
		AssignedName:      assignedName,
		PublicKey:         targetPub,
		ProviderPublicKey: providerPub,
	}
}

func (db *DbStore) RemoveAlias(alias *alias.Alias) {
	key := db.makeAliasKeyEntry(alias.PublicKey, alias.ProviderPublicKey)
	db.delete(key)
}

func (db *DbStore) GetAllAliases() []*alias.Alias {
	iter := db.db.NewIterator(util.BytesPrefix(aliasPrefix), nil)
	aliases := make([]*alias.Alias, 0, 10)
	for iter.Next() {
		key := iter.Key()
		val := iter.Value()
		if val != nil {
			targetPub, providerPub := db.recoverKeysFromAliasKeyField(key)
			aliases = append(aliases, &alias.Alias{
				AssignedName:      string(val),
				PublicKey:         targetPub,
				ProviderPublicKey: providerPub,
			})
		}
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		panic(err)
	}

	return aliases
}

func (db *DbStore) RemoveAllAliases() {
	iter := db.db.NewIterator(util.BytesPrefix(aliasPrefix), nil)
	for iter.Next() {
		db.delete(iter.Key())
	}

	iter.Release()
	if err := iter.Error(); err != nil {
		panic(err)
	}
}

// Close closes the database connection. It should be called upon server shutdown.
func (db *DbStore) Close() {
	db.db.Close()
}

func nonNilBytes(bz []byte) []byte {
	if bz == nil {
		return []byte{}
	}
	return bz
}

// NewDbStore returns new instance of a DbStore.
func NewDbStore(name string, dir string) (*DbStore, error) {
	dbPath := filepath.Join(dir, name+".db")
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, err
	}

	store := &DbStore{
		db: db,
	}
	return store, nil
}
