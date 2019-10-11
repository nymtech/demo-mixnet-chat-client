package chat_client

import (
	"encoding/base64"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/nymtech/nym-mixnet/config"
)

func (c *ChatClient) toChoosable(client config.ClientConfig) string {
	b64Key := base64.URLEncoding.EncodeToString(client.PubKey)
	b64ProviderKey := base64.URLEncoding.EncodeToString(client.Provider.PubKey)

	aliasedName := "<no alias>"
	possibleAlias := c.tryAliasStore(client.PubKey, client.Provider.PubKey)
	if possibleAlias != nil && possibleAlias.AssignedName != "" {
		aliasedName = possibleAlias.AssignedName
	}

	return fmt.Sprintf("%10s - (Pubkey) %s @[Provider] %s", aliasedName, b64Key, b64ProviderKey)
}

func (c *ChatClient) makeChoosables(recipients []config.ClientConfig) (map[string]config.ClientConfig, []string) {
	choosableRecipients := make(map[string]config.ClientConfig)
	options := make([]string, len(recipients)+1)
	for i, recipient := range recipients {
		choosableRecipient := c.toChoosable(recipient)
		options[i] = choosableRecipient
		choosableRecipients[choosableRecipient] = recipient // basically a mapping from the string back to original struct
	}

	options[len(recipients)] = refreshClientOption
	return choosableRecipients, options
}

func (c *ChatClient) chooseRecipient() (string, map[string]config.ClientConfig) {
	choosableRecipients, choosableOptions := c.makeChoosables(c.mixClient.Network.Clients)

	var chosenClientOption string
	prompt := &survey.Select{
		Message: "Choose another client to communicate with:",
		Options: choosableOptions,
	}
	if err := survey.AskOne(prompt, &chosenClientOption, nil); err == terminal.InterruptErr {
		c.Shutdown()
		return "", nil
	}
	// do not return actual client config as the chosen option might be "refresh"
	return chosenClientOption, choosableRecipients
}

// firstly choose our recipient, then create proper gui
// if this command line chat was to be used further down the line,
// I would have probably tried to implement the list selection in gocui,
// but using two frameworks is just way simpler in this case
func(c *ChatClient) getRecipient() (config.ClientConfig, error) {
	chosenClientOption := refreshClientOption
	var clientMapping map[string]config.ClientConfig
	for chosenClientOption == refreshClientOption {
		chosenClientOption, clientMapping = c.chooseRecipient()
		if err := c.mixClient.UpdateNetworkView(); err != nil {
			return config.ClientConfig{}, err
		}
	}

	if clientMapping == nil {
		return config.ClientConfig{}, ErrNoRecipient
	}

	return clientMapping[chosenClientOption], nil
}
