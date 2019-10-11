package chat_client

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/jroimartin/gocui"
	"github.com/nymtech/demo-mixnet-chat-client/chat-client/commands"
	"github.com/nymtech/demo-mixnet-chat-client/chat-client/commands/alias"
	"github.com/nymtech/demo-mixnet-chat-client/gui"
	"github.com/nymtech/demo-mixnet-chat-client/gui/layout"
	"github.com/nymtech/demo-mixnet-chat-client/message"
	"github.com/nymtech/demo-mixnet-chat-client/storage"
	"github.com/nymtech/demo-mixnet-chat-client/types"
	"github.com/nymtech/nym-mixnet/client"
	clientConfig "github.com/nymtech/nym-mixnet/client/config"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	refreshClientOption = "refresh the list of clients"

	// TODO: create new config.toml or include this in existing client config?
	defaultStoreFile = "chatstore"
	defaultStoreDir  = "chat-application"
)

var (
	ErrNoRecipient =  errors.New("no recipient was chosen")
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

	chatStore, err := storage.NewDbStore(chatStoreFile, filepath.Join(baseClientCfg.Client.FullMixAppsDir(), chatStoreDir))

	cc := &ChatClient{
		haltedCh:   make(chan struct{}),
		mixClient:  baseClient,
		chatStore:  chatStore,
		aliasCache: make(map[string]string),
	}

	return cc, nil
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
	availableCommandsString := ""
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

func (c *ChatClient) pollForMessages(g *gocui.Gui, sessionHalt <-chan struct{}) {
	time.Sleep(time.Second) // to make sure the main loop of gui starts first; TODO: better solution
	heartbeat := time.NewTicker(50 * time.Millisecond)
	// note: this does not perform any external queries,
	// it just checks the buffer of NetClient for whether it has any messages
	for {
		select {
		case <-c.haltedCh:
			return
		case <-sessionHalt:
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
		gui.WriteMessage(rawMsg, "You [cmd]", g)
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



func (c *ChatClient) startNewChatSession(sessionHalt chan struct{}) error {
	recipient, err := c.getRecipient()
	storedAlias := c.tryAliasStore(recipient.PubKey, recipient.Provider.PubKey)

	fullRecipientName := ""
	if storedAlias == nil || storedAlias.AssignedName == "" {
		fullRecipientName = base64.URLEncoding.EncodeToString(recipient.PubKey)
	} else {
		fullRecipientName = storedAlias.AssignedName
	}

	c.session = types.NewSession(recipient, fullRecipientName)

	if err != nil {
		return err
	}

	g, err := gui.CreateGUI()
	if err != nil {
		return err
	}
	defer g.Close()

	if err := c.initKeybindings(g); err != nil {
		return err
	}
	c.initCommands(g)

	// initial notices
	g.Update(func(g *gocui.Gui) error {
		b64Key := base64.URLEncoding.EncodeToString(c.mixClient.GetPublicKey().Bytes())
		gui.WriteNotice(fmt.Sprintf("Your public key is: %s Share it off channel with anyone you wish to communicate with.\n",
			b64Key,
		), g, "Reminder")

		gui.WriteNotice(fmt.Sprintf("You're currently sending messages to: %s\n",
			fullRecipientName,
		), g, "Reminder")
		c.showAvailableCommands(g)

		return c.updateSession(g)
	})

	go c.pollForMessages(g, sessionHalt)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}

	return nil
}

func (c *ChatClient) Run() error {
	if err := c.mixClient.Start(); err != nil {
		return err
	}

	var exitErr error = nil
	for exitErr == nil {
		sessionHalt := make(chan struct{})
		exitErr = c.startNewChatSession(sessionHalt)
	}

	if exitErr == ErrNoRecipient {
		c.Shutdown()
		return nil
	}
	return exitErr
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
