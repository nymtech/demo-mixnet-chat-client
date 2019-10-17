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
	loopix_cmd "github.com/nymtech/nym-mixnet/cmd/loopix-client/commands"
	"github.com/tav/golly/optparse"
)

func main() {
	var logo = `
	 _                      _           ____                       
	| |    ___   ___  _ __ (_)_  __    |  _ \  ___ _ __ ___   ___  
	| |   / _ \ / _ \| '_ \| \ \/ /____| | | |/ _ \ '_ \ _ \ / _ \ 
	| |___ (_) | (_) | |_) | |>  <_____| |_| |  __/ | | | | | (_) |
	|_____\___/ \___/| .__/|_/_/\_\    |____/ \___|_| |_| |_|\___/ 
			 |_|                                           
                                                                                       
		  `
	cmds := map[string]func([]string, string){
		"run":  cmd.RunCmd,
		"init": loopix_cmd.InitCmd,
	}
	info := map[string]string{
		"run":  "Run a persistent demo-chat client process",
		"init": "Initialise a base Loopix client",
	}
	optparse.Commands("demo-mixnet-chat-client", "0.0.2", cmds, info, logo)
}
