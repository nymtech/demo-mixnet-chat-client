package chat_client

import (
	"encoding/base64"
	"github.com/jroimartin/gocui"
	"github.com/nymtech/demo-mixnet-chat-client/chat-client/commands/alias"
	"github.com/nymtech/demo-mixnet-chat-client/gui/layout"
	"github.com/nymtech/demo-mixnet-chat-client/utils"
	"github.com/nymtech/nym-mixnet/sphinx"
)

func (c *ChatClient) resetView(v *gocui.View) error {
	v.Clear()
	return v.SetCursor(0, 0)
}

func (c *ChatClient) updateSendViewTitle(g *gocui.Gui) error {
	v, err := g.View(layout.InputViewName)
	if err != nil {
		return err
	}
	v.Title = "send to: " + c.session.RecipientAlias()
	return nil
}

func (c *ChatClient) makeAliasCacheKey(senderPublicKey, senderProviderPublicKey []byte) string {
	b64SenderKey := base64.URLEncoding.EncodeToString(senderPublicKey)
	b64SenderProviderKey := base64.URLEncoding.EncodeToString(senderProviderPublicKey)
	cacheEntryKey := b64SenderKey + b64SenderProviderKey
	return cacheEntryKey
}


// naming things is difficult...
func (c *ChatClient) recoverKeysFromCacheKey(key string) (*sphinx.PublicKey, *sphinx.PublicKey) {
	// due to both keys having same and constant length, we can just split the key in half
	// and due to it being in base64, hence containing only ASCII, we don't need to bother with runes and UTF8 encoding
	bKey := []byte(key)
	key1 := string(bKey[:len(bKey)/2])
	key2 := string(bKey[len(bKey)/2:])

	decodedKey1, err := base64.URLEncoding.DecodeString(key1)
	if err != nil {
		return nil, nil
	}
	decodedKey2, err := base64.URLEncoding.DecodeString(key2)
	if err != nil {
		return nil, nil
	}

	return utils.KeysFromBytes(decodedKey1, decodedKey2)
}


func (c *ChatClient) tryAliasStore(senderPublicKey, senderProviderPublicKey []byte) *alias.Alias {
	senderKey, senderProvKey := utils.KeysFromBytes(senderPublicKey, senderProviderPublicKey)
	if senderKey != nil && senderProvKey != nil {
		return c.chatStore.GetAlias(senderKey, senderProvKey)
	}
	return nil
}


func (c *ChatClient) defaultDisplayName(key []byte) string {
	b64Key := base64.URLEncoding.EncodeToString(key)
	return "??? - " + b64Key[:8] + "..."
}


func (c *ChatClient) getDisplayName(senderPublicKey, senderProviderPublicKey []byte) string {
	cacheEntryKey := c.makeAliasCacheKey(senderPublicKey, senderProviderPublicKey)
	displayName, ok := c.aliasCache[cacheEntryKey]
	if ok {
		return displayName
	}
	if storedAlias := c.tryAliasStore(senderPublicKey, senderProviderPublicKey); storedAlias != nil {
		// it's not in cache so update the cache
		if storedAlias.AssignedName != "" {
			cacheEntryKey := c.makeAliasCacheKey(senderPublicKey, senderProviderPublicKey)
			c.aliasCache[cacheEntryKey] = storedAlias.AssignedName
			return storedAlias.AssignedName
		}
	}
	return c.defaultDisplayName(senderPublicKey)
}
