package alias

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/jroimartin/gocui"
	"github.com/nymtech/demo-mixnet-chat-client/chat-client/commands"
	"github.com/nymtech/demo-mixnet-chat-client/gui"
	"github.com/nymtech/demo-mixnet-chat-client/types"
	"github.com/nymtech/nym-mixnet/sphinx"
)

const (
	noAlias          = "<none>"
	commandName      = "alias"
	showSubCommand   = "show"
	removeSubCommand = "remove"
	addSubCommand    = "add"
	allModifier      = "all"
)

var (
	ErrNotEnoughArguments = errors.New("alias command did not receive enough arguments")
	ErrInvalidArguments   = errors.New("alias command received invalid arguments")
	ErrMalformedRecipient = errors.New("malformed recipient data")
	ErrGeneric            = errors.New("could not handle alias command")

	forbiddenAliases = []string{
		noAlias,
		commandName,
		showSubCommand,
		removeSubCommand,
		addSubCommand,
		allModifier,
	}
)

type AliasStore interface {
	StoreAlias(alias *Alias)
	GetAlias(*sphinx.PublicKey, *sphinx.PublicKey) *Alias
	RemoveAlias(alias *Alias)
	RemoveAliasByKeys(*sphinx.PublicKey, *sphinx.PublicKey)
	GetAllAliasesByName(string) []*Alias
	GetAllAliases() []*Alias
	RemoveAllAliases()
}

type Alias struct {
	AssignedName      string
	PublicKey         *sphinx.PublicKey
	ProviderPublicKey *sphinx.PublicKey
}

func (a *Alias) String() string {
	b64Key := base64.URLEncoding.EncodeToString(a.PublicKey.Bytes())
	b64ProvKey := base64.URLEncoding.EncodeToString(a.ProviderPublicKey.Bytes())
	assignedName := a.AssignedName
	if assignedName == "" {
		assignedName = noAlias
	}
	return fmt.Sprintf("Alias: %s - Public Key: %s Provider's Public Key: %s", assignedName, b64Key, b64ProvKey)
}

type AliasCmd struct {
	g       *gocui.Gui
	store   AliasStore
	session *types.Session

	// TODO: possible set of subcommands? so separate explicit handlers for remove, add, show, etc
	//subCommands []commands.Command
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

func checkIfValidName(name string) bool {
	for _, invalidAlias := range forbiddenAliases {
		if name == invalidAlias {
			return false
		}
	}
	return true
}

func (a *AliasCmd) Name() string {
	return commandName
}

func (a *AliasCmd) Usage() string {
	usageString := "\n"
	usageString += fmt.Sprintf("\t/%s: \n", commandName)
	usageString += fmt.Sprintf("\t\t - /%s %s <aliased_named>\n", commandName, addSubCommand)
	usageString += fmt.Sprintf("\t\t - /%s %s <b64_public_key> <b64_provider_public_key> <aliased_name>\n", commandName, addSubCommand)
	usageString += fmt.Sprintf("\t\t - /%s %s\n", commandName, removeSubCommand)
	usageString += fmt.Sprintf("\t\t - /%s %s <b64_public_key> <b64_provider_public_key>\n", commandName, removeSubCommand)
	usageString += fmt.Sprintf("\t\t - /%s %s all\n", commandName, removeSubCommand)
	usageString += fmt.Sprintf("\t\t - /%s %s\n", commandName, showSubCommand)
	usageString += fmt.Sprintf("\t\t - /%s %s <aliased_name>\n", commandName, showSubCommand)
	usageString += fmt.Sprintf("\t\t - /%s %s all\n", commandName, showSubCommand)
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
			a.session.UpdateAlias("")
			return nil
		} else {
			return ErrMalformedRecipient
		}
	case 2:
		if args[1] == allModifier {
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
		if !checkIfValidName(args[1]) {
			return ErrInvalidArguments
		}
		currentPub, currentProvPub := a.getCurrentRecipientKeys()
		if currentPub != nil && currentProvPub != nil {
			alias := &Alias{
				AssignedName:      args[1],
				PublicKey:         currentPub,
				ProviderPublicKey: currentProvPub,
			}
			a.store.StoreAlias(alias)
			a.session.UpdateAlias(args[1])
			return nil
		} else {
			return errors.New("malformed recipient data")
		}
	case 4:
		if !checkIfValidName(args[3]) {
			return ErrInvalidArguments
		}
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
		currentPub, currentProvPub := a.getCurrentRecipientKeys()
		if currentPub != nil && currentProvPub != nil {
			currentAlias := a.store.GetAlias(currentPub, currentProvPub)
			gui.WriteInfo(currentAlias.String()+"\n", a.g, "alias_info")
			return nil
		}
		return ErrMalformedRecipient
	case 2:
		var aliases []*Alias
		if args[1] == allModifier {
			aliases = a.store.GetAllAliases()
			if len(aliases) == 0 {
				gui.WriteInfo("no aliases assigned", a.g, "alias_info")
			}
		} else {
			aliases = a.store.GetAllAliasesByName(args[1])
			if len(aliases) == 0 {
				gui.WriteInfo(fmt.Sprintf("no clients with alias: %s", args[1]), a.g, "alias_info")
			}
		}

		for _, alias := range aliases {
			gui.WriteInfo(alias.String()+"\n", a.g, "alias_info")
		}
		return nil

	default:
		return ErrInvalidArguments
	}
}

func (a *AliasCmd) Handle(args []string) error {
	// first element in the slice is the name of the command itself and always exists
	if len(args) == 1 {
		return ErrNotEnoughArguments
	}
	// sanity check
	if args[0] != commandName {
		return fmt.Errorf("invalid handler called. Expected: %s. got: %s", a.Name(), args[0])
	}
	switch args[1] {
	case addSubCommand:
		return a.handleAdd(args[1:])
	case removeSubCommand:
		return a.handleRemove(args[1:])
	case showSubCommand:
		return a.handleShow(args[1:])
	default:
		fmt.Println(args[1])
		return ErrInvalidArguments
	}
}

// AliasCommand creates new instance of an AliasCommand
// Each equivalent function for each command will take required context to resolve the command
func AliasCommand(g *gocui.Gui, store AliasStore, session *types.Session) commands.Command {
	return &AliasCmd{
		g:       g,
		store:   store,
		session: session,
	}
}
