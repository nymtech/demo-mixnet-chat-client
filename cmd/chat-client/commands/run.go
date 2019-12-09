// Copyright 2019 The Nym Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"fmt"
	"os"

	chat_client "github.com/nymtech/demo-mixnet-chat-client/chat-client"
	clientConfig "github.com/nymtech/nym-mixnet/client/config"
	"github.com/nymtech/nym-mixnet/helpers"
	"github.com/tav/golly/optparse"
)

const (
	defaultID = "Client"
)

//nolint: lll
func RunCmd(args []string, usage string) {
	opts := newOpts("run [OPTIONS]", usage)
	id := opts.Flags("--id").Label("ID").String("Id of the nym mixnet client we will use to run", defaultID)
	customConfigPath := opts.Flags("--customCfg").Label("CUSTOMCFG").String("Path to custom configuration file of the mixnet client", "")

	params := opts.Parse(args)
	if len(params) != 0 {
		opts.PrintUsage()
		os.Exit(1)
	}

	var configPath string
	var err error
	if len(*customConfigPath) > 0 {
		configPath = *customConfigPath
	} else {
		configPath, err = clientConfig.DefaultConfigPath(*id)
		if err != nil {
			panic(err)
		}
	}

	cfgExists, err := helpers.DirExists(configPath)
	if !cfgExists || err != nil {
		fmt.Fprintf(os.Stderr, "The configuration file at %v does not seem to exist\n", configPath)
		os.Exit(1)
	}

	cfg, err := clientConfig.LoadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not load the config file: %v\n", err)
		os.Exit(1)
	}

	chatClient, err := chat_client.New(cfg)
	if err != nil {
		panic(err)
	}

	if err := chatClient.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run client instance: %v\n", err)
		os.Exit(-1)
	}

	chatClient.Wait()
}

func newOpts(command string, usage string) *optparse.Parser {
	return optparse.New("Usage: chat-client " + command + "\n\n  " + usage + "\n")
}
