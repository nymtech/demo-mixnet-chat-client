package chat_client

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/golang/protobuf/proto"
	"github.com/jroimartin/gocui"
	"github.com/nymtech/demo-mixnet-chat-client/chat-client/commands"
	"github.com/nymtech/demo-mixnet-chat-client/chat-client/commands/alias"
	"github.com/nymtech/demo-mixnet-chat-client/gui"
	"github.com/nymtech/demo-mixnet-chat-client/gui/layout"
	"github.com/nymtech/demo-mixnet-chat-client/message"
	"github.com/nymtech/demo-mixnet-chat-client/storage"
	"github.com/nymtech/demo-mixnet-chat-client/types"
	"github.com/nymtech/demo-mixnet-chat-client/utils"
	"github.com/nymtech/nym-mixnet/client"
	clientConfig "github.com/nymtech/nym-mixnet/client/config"
	"github.com/nymtech/nym-mixnet/config"
	"github.com/nymtech/nym-mixnet/sphinx"
	"strings"
	"sync"
	"time"
)

const (
	refreshClientOption = "refresh the list of clients"
)

// str to func <- command
// some local file with mapping of (pubkey, provider) - human readable id

const (
	// TODO: create new config.toml or include this in existing client config?
	defaultStoreFile = "chatstore"
	defaultStoreDir  = ""
)

type ChatClient struct {
	session           *types.Session
	availableCommands []commands.Command
	chatStore         storage.ChatStore
	// so we wouldn't need to load it from file storage on every single received message
	aliasCache map[string]string
	mixClient  *client.NetClient
	haltedCh   chan struct{}
	haltOnce   sync.Once
}

func New(baseClientCfg *clientConfig.Config) (*ChatClient, error) {
	baseClient, err := client.NewClient(baseClientCfg)
	if err != nil {
		return nil, err
	}

	// TODO: configurable?
	chatStoreFile := defaultStoreFile
	chatStoreDir := defaultStoreDir

	chatStore, err := storage.NewDbStore(chatStoreFile, chatStoreDir)

	cc := &ChatClient{
		haltedCh:   make(chan struct{}),
		mixClient:  baseClient,
		chatStore:  chatStore,
		aliasCache: make(map[string]string),
	}

	return cc, nil
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

// checkCache makes sure that any entry present in the cache still exists in the store
// only called on new command execution
func (c *ChatClient) checkCache() {
	for k, v := range c.aliasCache {
		clientKey, clientProviderKey := c.recoverKeysFromCacheKey(k)
		storedAlias := c.chatStore.GetAlias(clientKey, clientProviderKey)
		if storedAlias == nil || storedAlias.AssignedName != v {
			delete(c.aliasCache, k)
		}
	}
}

func (c *ChatClient) updateSession(g *gocui.Gui) error {
	if err := c.updateSendViewTitle(g); err != nil {
		return err
	}
	c.checkCache()

	return nil
}

func (c *ChatClient) showAvailableCommands(g *gocui.Gui) {
	availableCommandsString := "\n"
	for _, cmd := range c.availableCommands {
		availableCommandsString += cmd.Usage()
		availableCommandsString += "\n"
	}

	gui.WriteInfo(availableCommandsString, g, "Available commands")
}

func (c *ChatClient) parseCommand(g *gocui.Gui, cmd string) error {
	cmd = strings.TrimPrefix(cmd, "/") // remove the backslach cmd prefix
	if len(cmd) == 0 {
		return errors.New("no valid command provided")
	}
	args := strings.Split(strings.TrimSpace(cmd), " ")

	mainCmd := args[0]
	for _, command := range c.availableCommands {
		if command.Name() == mainCmd {
			err := command.Handle(args)
			if err := c.updateSession(g); err != nil {
				return err
			}
			return err
		}
	}

	gui.WriteNotice(fmt.Sprintf("Command: %v does not exist\n", mainCmd), g, "error")
	c.showAvailableCommands(g)
	return nil
}

func (c *ChatClient) resetView(v *gocui.View) error {
	v.Clear()
	return v.SetCursor(0, 0)
}

func (c *ChatClient) parseReceivedMessages(msgs [][]byte) []*message.ChatMessage {
	parsedMsgs := make([]*message.ChatMessage, 0, len(msgs))
	if msgs == nil {
		return parsedMsgs
	}
	for _, msg := range msgs {
		if msg != nil {
			parsedMsg := &message.ChatMessage{}
			if err := proto.Unmarshal(msg, parsedMsg); err == nil {
				parsedMsgs = append(parsedMsgs, parsedMsg)
			}
		}
	}

	// for now completely ignore ordering
	return parsedMsgs
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

func (c *ChatClient) tryAliasStore(senderPublicKey, senderProviderPublicKey []byte) *alias.Alias {
	senderKey, senderProvKey := utils.KeysFromBytes(senderPublicKey, senderProviderPublicKey)
	if senderKey != nil && senderProvKey != nil {
		return c.chatStore.GetAlias(senderKey, senderProvKey)
	}
	return nil
}

func (c *ChatClient) pollForMessages(g *gocui.Gui) {
	time.Sleep(time.Second) // to make sure the main loop of gui starts first; TODO: better solution
	heartbeat := time.NewTicker(50 * time.Millisecond)
	// note: this does not perform any external queries,
	// it just checks the buffer of NetClient for whether it has any messages
	for {
		select {
		case <-c.haltedCh:
			return
		case <-heartbeat.C:
			msgs := c.mixClient.GetReceivedMessages()
			if len(msgs) > 0 {
				parsedMsgs := c.parseReceivedMessages(msgs)
				for _, msg := range parsedMsgs {
					// for now ignore any data in the message apart from the content
					content := string(msg.Content)
					if !strings.HasSuffix(content, "\n") {
						content += "\n"
					}
					gui.WriteMessage(content, c.getDisplayName(msg.SenderPublicKey, msg.SenderProviderPublicKey), g)
				}
			}
		}
	}
}

func (c *ChatClient) createMessagePayload(msg string) ([]byte, error) {
	protoPayload := &message.ChatMessage{
		Content:                 []byte(msg),
		SenderPublicKey:         c.mixClient.GetPublicKey().Bytes(),
		SenderProviderPublicKey: c.mixClient.Provider.PubKey,
		MessageNonce:            c.session.IncrementNonce(),
		SenderTimestamp:         time.Now().UnixNano(),
		Signature:               nil, // will be done later
	}

	return proto.Marshal(protoPayload)
}

func (c *ChatClient) handleSend(g *gocui.Gui, v *gocui.View) error {
	if v.Name() != layout.InputViewName {
		return fmt.Errorf("invalid view. Expected: %s, got: %s", layout.InputViewName, v.Name())
	}
	defer func() {
		if err := c.resetView(v); err != nil {
			// log...
		}
	}()

	v.Rewind()
	rawMsg := v.Buffer()
	if strings.HasPrefix(rawMsg, "/") {
		return c.parseCommand(g, rawMsg)
	}

	chatMsg, err := c.createMessagePayload(rawMsg)
	if err != nil {
		// todo: handle
	}
	if err := c.mixClient.SendMessage(chatMsg, c.session.Recipient()); err != nil {
		// log
		gui.WriteNotice("Could not send message", g, "ERROR")
	}

	// todo: parse our message before writing it to view (like add prefix of "you", etc) + do some validation
	msg := rawMsg

	gui.WriteMessage(msg, "You", g)

	return nil
}

func (c *ChatClient) initKeybindings(g *gocui.Gui) error {

	if err := g.SetKeybinding(layout.InputViewName, gocui.KeyEnter, gocui.ModNone, c.handleSend); err != nil {
		return err
	}

	return nil
}

func (c *ChatClient) initCommands(g *gocui.Gui) {
	c.availableCommands = []commands.Command{
		alias.AliasCommand(g, c.chatStore, c.session),
	}
}

func (c *ChatClient) updateSendViewTitle(g *gocui.Gui) error {
	v, err := g.View(layout.InputViewName)
	if err != nil {
		return err
	}
	v.Title = "send to: " + c.session.RecipientAlias()
	return nil
}

func (c *ChatClient) Start() error {
	if err := c.mixClient.Start(); err != nil {
		return err
	}

	// firstly choose our recipient, then create proper gui
	// if this command line chat was to be used further down the line,
	// I would have probably tried to implement the list selection in gocui,
	// but using two frameworks is just way simpler in this case

	chosenClientOption := refreshClientOption
	var clientMapping map[string]config.ClientConfig
	for chosenClientOption == refreshClientOption {
		chosenClientOption, clientMapping = c.chooseRecipient()
		if err := c.mixClient.UpdateNetworkView(); err != nil {
			return fmt.Errorf("failed to obtain recipient: %v", err)
		}
	}

	recipient := clientMapping[chosenClientOption]
	storedAlias := c.tryAliasStore(recipient.PubKey, recipient.Provider.PubKey)

	c.session = types.NewSession(recipient, c.getDisplayName(recipient.PubKey, recipient.Provider.PubKey))

	g, err := gui.CreateGUI()
	if err != nil {
		return err
	}
	defer g.Close()

	if err := c.initKeybindings(g); err != nil {
		return err
	}
	c.initCommands(g)

	b64Key := base64.URLEncoding.EncodeToString(c.mixClient.GetPublicKey().Bytes())
	gui.WriteNotice(fmt.Sprintf("Your public key is: %s Share it off channel with anyone you wish to communicate with.\n",
		b64Key,
	), g, "Reminder")

	fullRecipientName := ""
	if storedAlias == nil {
		fullRecipientName = base64.URLEncoding.EncodeToString(recipient.PubKey)
	} else {
		fullRecipientName = storedAlias.String()
	}
	gui.WriteNotice(fmt.Sprintf("You're currently sending messages to: %s\n",
		fullRecipientName,
	), g, "Reminder")

	go c.pollForMessages(g)

	// TODO: write available commands
	c.showAvailableCommands(g)
	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}
	// TODO: allow to start over with new recipient?

	return nil
}

// Wait waits till the client is terminated for any reason.
func (c *ChatClient) Wait() {
	<-c.haltedCh
}

// Shutdown cleanly shuts down a given client instance.
func (c *ChatClient) Shutdown() {
	c.haltOnce.Do(func() { c.halt() })
}

// calls any required cleanup code
func (c *ChatClient) halt() {
	//c.log.Infof("Starting graceful shutdown")
	// close any listeners, free resources, etc
	c.mixClient.Shutdown()

	close(c.haltedCh)
}

func toChoosable(client config.ClientConfig) string {
	b64Key := base64.URLEncoding.EncodeToString(client.PubKey)
	b64ProviderKey := base64.URLEncoding.EncodeToString(client.Provider.PubKey)
	// while normally it's unsafe to directly index string, it's safe here
	// as id is guaranteed to only hold ascii characters due to being b64 encoding of the key
	return fmt.Sprintf("ID: %s\t@[Provider]\t%s", b64Key, b64ProviderKey)
}

func makeChoosables(clients []config.ClientConfig) (map[string]config.ClientConfig, []string) {
	choosableClients := make(map[string]config.ClientConfig)
	options := make([]string, len(clients)+1)
	for i, client := range clients {
		choosableClient := toChoosable(client)
		options[i] = choosableClient
		choosableClients[choosableClient] = client // basically a mapping from the string back to original struct
	}

	options[len(clients)] = refreshClientOption
	return choosableClients, options
}

func (c *ChatClient) chooseRecipient() (string, map[string]config.ClientConfig) {
	choosableClients, choosableOptions := makeChoosables(c.mixClient.Network.Clients)

	var chosenClientOption string
	prompt := &survey.Select{
		Message: "Choose another client to communicate with:",
		Options: choosableOptions,
	}
	if err := survey.AskOne(prompt, &chosenClientOption, nil); err == terminal.InterruptErr {
		// we got an interrupt so we're killing whole client
		//c.log.Warningf("Received an interrupt - stopping entire client")
		c.Shutdown()
		return "", nil
	}
	// do not return actual client config as the chosen option might be "refresh"
	return chosenClientOption, choosableClients
}
