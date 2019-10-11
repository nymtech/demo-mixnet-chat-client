package utils

import "github.com/nymtech/nym-mixnet/sphinx"

func KeysFromBytes(targetKey, targetProvKey []byte) (*sphinx.PublicKey, *sphinx.PublicKey) {
	if targetKey == nil || targetProvKey == nil {
		return nil, nil
	}
	targetPub := new(sphinx.PublicKey)
	targetProvPub := new(sphinx.PublicKey)

	if err := targetPub.UnmarshalBinary(targetKey); err != nil {
		return nil, nil
	}
	if err := targetProvPub.UnmarshalBinary(targetProvKey); err != nil {
		return nil, nil
	}

	return targetPub, targetProvPub
}
