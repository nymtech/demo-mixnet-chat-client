package chat_client

import (
	"encoding/base64"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/nymtech/nym-mixnet/client"
	"github.com/nymtech/nym-mixnet/config"
	"sync"
	clientConfig "github.com/nymtech/nym-mixnet/client/config"
)

// str to func <- command
// some local file with mapping of (pubkey, provider) - human readable id

type ChatClient struct {
	mixClient *client.NetClient
	haltedCh  chan struct{}
	haltOnce  sync.Once
	//log      *logrus.Logger
}

func New(baseClientCfg *clientConfig.Config) (*ChatClient, error) {
	baseClient, err := client.NewClient(baseClientCfg)
	if err != nil {
		return nil, err
	}
	cc := &ChatClient{
		haltedCh: make(chan struct{}),
		mixClient: baseClient,
	}

	return cc, nil
}

func (c *ChatClient) Start() error {
	if err := c.mixClient.Start(); err != nil {
		return err
	}

	go c.startInputRoutine()
	return nil
}

// Wait waits till the client is terminated for any reason.
func (c *ChatClient) Wait() {
	<-c.haltedCh
}

// Shutdown cleanly shuts down a given client instance.
// TODO: create daemon to call this upon sigterm or something
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

// THE BELOW CODE WAS JUST COPIED FROM THE ORIGINAL CLIENT CODE, IT WILL BE IMPROVED LATER ON,
// BY FOR EXAMPLE MOVING LOGGER UP TO THE CHATCLIENT, BETTER 'GUI' ETC

func toChoosable(client config.ClientConfig) string {
	b64Key := base64.URLEncoding.EncodeToString(client.PubKey)
	b64ProviderKey := base64.URLEncoding.EncodeToString(client.Provider.PubKey)
	// while normally it's unsafe to directly index string, it's safe here
	// as id is guaranteed to only hold ascii characters due to being b64 encoding of the key
	return fmt.Sprintf("ID: %s\t@[Provider]\t%s", b64Key, b64ProviderKey)
}

func makeChoosables(clients []config.ClientConfig) (map[string]config.ClientConfig, []string) {
	choosableClients := make(map[string]config.ClientConfig)
	options := make([]string, len(clients))
	for i, client := range clients {
		choosableClient := toChoosable(client)
		options[i] = choosableClient
		choosableClients[choosableClient] = client // basically a mapping from the string back to original struct
	}
	return choosableClients, options
}

func shouldStopInput(msg string) bool {
	quitMessages := []string{
		"quit",
		"/q",
		":q",
		":q!",
		"exit",
	}

	for _, qm := range quitMessages {
		if qm == msg {
			return true
		}
	}

	return false
}

func (c *ChatClient) startInputRoutine() {

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
		return
	}

	chosenClient := choosableClients[chosenClientOption]

	for {
		select {
		case <-c.haltedCh:
			c.Shutdown()
			return
		default:
		}
		messageToSend := ""
		b64Key := base64.URLEncoding.EncodeToString(chosenClient.GetPubKey())
		prompt := &survey.Input{
			Message: fmt.Sprintf("Type in a message to send to %s...", b64Key),
		}
		if err := survey.AskOne(prompt, &messageToSend); err == terminal.InterruptErr {
			// we got an interrupt so we're killing whole client
			//c.log.Warningf("Received an interrupt - stopping entire client")
			c.Shutdown()
			return
		}
		if shouldStopInput(messageToSend) {
			//c.log.Warningf("Received a stop signal. Stopping the input routine")
			return
		}

		//c.log.Infof("Sending: %v to %v", messageToSend, chosenClient.GetId())
		if err := c.mixClient.SendMessage(messageToSend, chosenClient); err != nil {
			//c.log.Errorf("Failed to send %v to %x...: %v", messageToSend, chosenClient.GetPubKey()[:8], err)
		}
	}
}
