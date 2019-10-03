// Copyright 2019 The Loopix-Messaging Authors
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

package main

import (
	cmd "github.com/nymtech/demo-mixnet-chat-client/cmd/chat-client/commands"
	"github.com/tav/golly/optparse"
)

func main() {
	var logo = `
  ____                              ____ _           _         ____ _ _            _   
 |  _ \  ___ _ __ ___   ___        / ___| |__   __ _| |_      / ___| (_) ___ _ __ | |_ 
 | | | |/ _ \ '_ \ _ \ / _ \ _____| |   | '_ \ / _\ | __|____| |   | | |/ _ \ '_ \| __|
 | |_| |  __/ | | | | | (_) |_____| |___| | | | (_| | |______| |___| | |  __/ | | | |_ 
 |____/ \___|_| |_| |_|\___/       \____|_| |_|\__,_|\__|     \____|_|_|\___|_| |_|\__|
                                                                                       
		  `
	cmds := map[string]func([]string, string){
		"run":  cmd.RunCmd,
	}
	info := map[string]string{
		"run": "Run a persistent demo-chat client process",
	}
	optparse.Commands("demo-mixnet-chat-client", "0.0.1", cmds, info, logo)
}
