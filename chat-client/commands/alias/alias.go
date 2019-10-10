package alias

import (
	"encoding/base64"
	"errors"
	"github.com/jroimartin/gocui"
	"github.com/nymtech/demo-mixnet-chat-client/chat-client/commands"
	"github.com/nymtech/demo-mixnet-chat-client/types"
	"github.com/nymtech/nym-mixnet/sphinx"
)

var (
	ErrNotEnoughArguments = errors.New("alias command did not receive enough arguments")
	ErrInvalidArguments = errors.New("alias command received invalid arguments")
	ErrGeneric = errors.New("could not handle alias command")

	forbiddenAliases = []string{
		"all",
	}
)

type Alias struct {
	AssignedName      string
	PublicKey         *sphinx.PublicKey
	ProviderPublicKey *sphinx.PublicKey
}

type AliasStore interface {
	StoreAlias(alias *Alias)
	GetAlias(*sphinx.PublicKey, *sphinx.PublicKey) *Alias
	RemoveAlias(alias *Alias)
	RemoveAliasByKeys(*sphinx.PublicKey, *sphinx.PublicKey)
	GetAllAliases() []*Alias
	RemoveAllAliases()
}

type AliasCmd struct {
	g     *gocui.Gui
	store AliasStore
	session *types.Session
}


func (a *AliasCmd) keysFromBytes(targetKey, targetProvKey []byte) (*sphinx.PublicKey, *sphinx.PublicKey) {
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

func (a *AliasCmd) getCurrentRecipientKeys() (*sphinx.PublicKey, *sphinx.PublicKey) {
	return a.keysFromBytes(a.session.Recipient().PubKey, a.session.Recipient().Provider.PubKey)
}


func (a *AliasCmd) getTargetKeysFromStrings(targetKey, targetProvKey string) (*sphinx.PublicKey, *sphinx.PublicKey) {
	targetKeyB, err := base64.URLEncoding.DecodeString(targetKey)
	if err != nil {
		return nil, nil
	}
	targetProvKeyB, err := base64.URLEncoding.DecodeString(targetProvKey)
	if err != nil {
		return nil, nil
	}

	return a.keysFromBytes(targetKeyB, targetProvKeyB)
}



func (a *AliasCmd) Name() string {
	return "alias"
}

func (a *AliasCmd) Usage() string {
	usageString := "\n"
	usageString += "\t/alias: \n"
	usageString += "\t\t - /alias add <aliased_named>\n"
	usageString += "\t\t - /alias add <b64_public_key> <b64_provider_public_key> <aliased_name>\n"
	usageString += "\t\t - /alias remove\n"
	usageString += "\t\t - /alias remove <b64_public_key> <b64_provider_public_key>\n"
	usageString += "\t\t - /alias remove all\n"
	usageString += "\t\t - /alias show\n"
	usageString += "\t\t - /alias show <aliased_name>\n"
	usageString += "\t\t - /alias show all\n"
	return usageString
}

// we expect the following:
// just `remove` which will remove the alias for current recipient
// `remove <pubkey> <provider_pubkey>` which will remove alias for the client
// `remove all` which will remove all aliases
func (a *AliasCmd) handleRemove(args []string) error {
	// first element in the slice is the name of the command itself and always exists
	switch len(args) {
	case 1:
		// we remove it for current entry
		currentPub, currentProvPub := a.getCurrentRecipientKeys()
		if currentPub != nil && currentProvPub != nil {
			a.store.RemoveAliasByKeys(currentPub, currentProvPub)
			return nil
		} else {
			return errors.New("malformed recipient data")
		}
	case 2:
		if args[1] == "all" {
			a.store.RemoveAllAliases()
			return nil
		} else {
			return ErrInvalidArguments
		}
	case 3:
		targetKey, targetProvKey := a.getTargetKeysFromStrings(args[1], args[2])
		if targetKey != nil && targetProvKey != nil {
			a.store.RemoveAliasByKeys(targetKey, targetProvKey)
			return nil
		} else {
			return ErrInvalidArguments
		}
	default:
		return ErrInvalidArguments
	}
}

// we expect the following:
// `add <alias>` which will create alias for the current recipient
// `add <pubkey> <provider_pubkey> <alias>` which will create alias for the specified recipient. note: both keys have to be provided in base64
func (a *AliasCmd) handleAdd(args []string) error {
	// first element in the slice is the name of the command itself and always exists
	switch len(args) {
	case 1:
		return ErrNotEnoughArguments
	case 2:
		currentPub, currentProvPub := a.getCurrentRecipientKeys()
		if currentPub != nil && currentProvPub != nil {
			alias := &Alias{
				AssignedName:      args[1],
				PublicKey:         currentPub,
				ProviderPublicKey: currentProvPub,
			}
			a.store.StoreAlias(alias)
			return nil
		} else {
			return errors.New("malformed recipient data")
		}
	case 4:
		targetKey, targetProvKey := a.getTargetKeysFromStrings(args[1], args[2])
		if targetKey != nil && targetProvKey != nil {
			alias := &Alias{
				AssignedName:      args[3],
				PublicKey:         targetKey,
				ProviderPublicKey: targetProvKey,
			}
			a.store.StoreAlias(alias)
			return nil
		} else {
			return ErrInvalidArguments
		}
	default:
		return ErrInvalidArguments
	}
}

// we expect the following:
// `show` which will print all details for current recipient
// `show <alias>` which will print all details for the specified alias
// `show all` which will print details for all stored aliases
func (a *AliasCmd) handleShow(args []string) error {
	// first element in the slice is the name of the command itself and always exists
	switch len(args) {
	case 1:

	default:
		return ErrInvalidArguments
	}
}

func (a *AliasCmd) Handle(args []string) error {
	// first element in the slice is the name of the command itself and always exists
	if len(args) == 1 {
		return ErrNotEnoughArguments
	}
	switch args[1] {
	case "add", "new":
		return a.handleAdd(args[1:])
	case "rm", "remove", "delete":
		return a.handleRemove(args[1:])
	case "show", "display":
		return a.handleShow(args[1:])
	default:
		return ErrInvalidArguments
	}
}

// AliasCommand creates new instance of an AliasCommand
// Each equivalent function for each command will take required context to resolve the command
func AliasCommand(g *gocui.Gui, store AliasStore, session *types.Session) commands.Command {
	return &AliasCmd{
		g:     g,
		store: store,
		session:session,
	}
}