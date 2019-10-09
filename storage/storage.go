// storage for everything required by the chat-client. From persistent local state to client aliases
package storage

import (
	"github.com/nymtech/demo-mixnet-chat-client/chat-client/alias"
	"github.com/nymtech/nym-mixnet/sphinx"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/util"
	"path/filepath"
)

var (
	aliasPrefix = []byte("ALIAS")
)

// Store represents all data required to interact with the storage.
type Store struct {
	db *leveldb.DB
}

// get gets the value corresponding to particular key. Returns nil if it doesn't exist.
func (s *Store) get(key []byte) []byte {
	key = nonNilBytes(key)
	res, err := s.db.Get(key, nil)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil
		}
		panic(err)
	}
	return res
}

// set sets particular key value pair.
func (s *Store) set(key []byte, value []byte) {
	key = nonNilBytes(key)
	value = nonNilBytes(value)
	if err := s.db.Put(key, value, nil); err != nil {
		panic(err)
	}
}

// delete removes particular key value pair.
func (s *Store) delete(key []byte) {
	key = nonNilBytes(key)
	if err := s.db.Delete(key, nil); err != nil {
		panic(err)
	}
}

// --------- ALIAS RELATED -----------

// Each alias corresponds to the tuple of user's public key and the public key of it's provider
// each entry follows the structure of: [ ALIAS_PREFIX || PUBLIC_KEY || PROVIDER_PUBLIC_KEY ] -- ALIAS

func (s *Store) makeAliasKeyEntry(targetPub, providerPub *sphinx.PublicKey) []byte {
	if targetPub == nil || providerPub == nil {
		return []byte{}
	}
	key := make([]byte, len(aliasPrefix)+2*sphinx.PublicKeySize)
	i := copy(key, aliasPrefix)
	i += copy(key[i:], targetPub.Bytes())
	copy(key[i:], providerPub.Bytes())
	return key
}

func (s *Store) recoverKeysFromAliasKeyField(key []byte) (*sphinx.PublicKey, *sphinx.PublicKey) {
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

func (s *Store) StoreAlias(alias *alias.Alias) {
	key := s.makeAliasKeyEntry(alias.PublicKey, alias.ProviderPublicKey)
	// even if the entry already exists, overwrite it
	s.set(key, []byte(alias.AssignedName))
}

func (s *Store) GetAlias(targetPub, providerPub *sphinx.PublicKey) *alias.Alias {
	key := s.makeAliasKeyEntry(targetPub, providerPub)
	aliasB := s.get(key)
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

func (s *Store) RemoveAlias(alias *alias.Alias) {
	key := s.makeAliasKeyEntry(alias.PublicKey, alias.ProviderPublicKey)
	s.delete(key)
}

func (s *Store) GetAllAliases() []*alias.Alias {
	iter := s.db.NewIterator(util.BytesPrefix(aliasPrefix), nil)
	aliases := make([]*alias.Alias, 0, 10)
	for iter.Next() {
		key := iter.Key()
		val := iter.Value()
		if val != nil {
			targetPub, providerPub := s.recoverKeysFromAliasKeyField(key)
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

func (s *Store) RemoveAllAliases() {
	iter := s.db.NewIterator(util.BytesPrefix(aliasPrefix), nil)
	for iter.Next() {
		s.delete(iter.Key())
	}

	iter.Release()
	if err := iter.Error(); err != nil {
		panic(err)
	}
}

// Close closes the database connection. It should be called upon server shutdown.
func (s *Store) Close() {
	s.db.Close()
}

func nonNilBytes(bz []byte) []byte {
	if bz == nil {
		return []byte{}
	}
	return bz
}

// New returns new instance of a store.
func New(name string, dir string) (*Store, error) {
	dbPath := filepath.Join(dir, name+".db")
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, err
	}

	store := &Store{
		db: db,
	}
	return store, nil
}
