package chat_client

import (
	"encoding/base64"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/golang/protobuf/proto"
	"github.com/jroimartin/gocui"
	"github.com/nymtech/demo-mixnet-chat-client/gui"
	"github.com/nymtech/demo-mixnet-chat-client/gui/layout"
	"github.com/nymtech/demo-mixnet-chat-client/message"
	"github.com/nymtech/nym-mixnet/client"
	clientConfig "github.com/nymtech/nym-mixnet/client/config"
	"github.com/nymtech/nym-mixnet/config"
	"strings"
	"sync"
	"time"
)

const (
	refreshClientOption = "refresh the list of clients"
)

// str to func <- command
// some local file with mapping of (pubkey, provider) - human readable id

type ChatClient struct {
	recipient config.ClientConfig
	mixClient *client.NetClient
	haltedCh  chan struct{}
	haltOnce  sync.Once
	sessionNonce int64
}

func New(baseClientCfg *clientConfig.Config) (*ChatClient, error) {
	baseClient, err := client.NewClient(baseClientCfg)
	if err != nil {
		return nil, err
	}
	cc := &ChatClient{
		haltedCh:  make(chan struct{}),
		mixClient: baseClient,
	}

	return cc, nil
}

func (c *ChatClient) parseCommand(cmd string) error {
	return nil
}

func (c *ChatClient) resetView(v *gocui.View) error {
	v.Clear()
	return v.SetCursor(0, 0)
}

func (c *ChatClient) parseMessages(msgs [][]byte) []*message.ChatMessage {
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
				parsedMsgs := c.parseMessages(msgs)
				for _, msg := range parsedMsgs {
					// for now ignore any data in the message apart from the content
					content := string(msg.Content)
					b64SenderKey := base64.URLEncoding.EncodeToString(msg.SenderPublicKey)
					if !strings.HasSuffix(content, "\n") {
						content += "\n"
					}
					c.updateSendViewTitle(g, b64SenderKey)
					gui.WriteMessage(content, "??? - " + b64SenderKey[:8] + "...", g)
				}
			}
		}
	}
}

func (c *ChatClient) getNonce() int64 {
	return c.sessionNonce + 1
}

func (c *ChatClient) createMessagePayload(msg string) ([]byte, error) {
	protoPayload := &message.ChatMessage{
		Content:                 []byte(msg),
		SenderPublicKey:         c.mixClient.GetPublicKey().Bytes(),
		SenderProviderPublicKey: c.mixClient.Provider.PubKey,
		MessageNonce:            c.getNonce(),
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
		gui.WriteNotice("Trying to parse entered command...", g, "debug_info")
		return c.parseCommand(rawMsg)
	}

	chatMsg, err := c.createMessagePayload(rawMsg)
	if err != nil {
		// todo: handle
	}
	if err := c.mixClient.SendMessage(chatMsg, c.recipient); err != nil {
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

func (c *ChatClient) updateSendViewTitle(g *gocui.Gui, newTitle string) error {
	v, err := g.View(layout.InputViewName)
	if err != nil {
		return err
	}
	v.Title = "send to: " + newTitle
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

	c.recipient = clientMapping[chosenClientOption]

	g, err := gui.CreateGUI()
	if err != nil {
		return err
	}
	defer g.Close()

	if err := c.initKeybindings(g); err != nil {
		return err
	}

	b64Key := base64.URLEncoding.EncodeToString(c.mixClient.GetPublicKey().Bytes())
	gui.WriteNotice(fmt.Sprintf("Your public key is: %s Share it off channel with anyone you wish to communicate with.\n",
		b64Key,
	), g, "Reminder")

	recipientName := c.recipient.Id
	// TODO: lookup aliases file and change if necessary
	gui.WriteNotice(fmt.Sprintf("You're currently sending messages to: %s\n",
		recipientName,
	), g, "Reminder")

	go c.pollForMessages(g)

	// TODO: write available commands

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}

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

