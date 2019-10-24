# demo-mixnet-chat-client

A simple chat client which sends its traffic through the nym-mixnet

On this branch it uses an Electron frontend for user interaction. Note that if you want to build the binary yourself, you need to modify the following line in `package.json`:

```bash
"prepare-loopix": "cwd=$(pwd) && cd $HOME/workspace/nym-related/nym-mixnet/cmd/loopix-client && go build -o $cwd/dist/loopix-client"
```

so that the path points to the sourcecode of nym-mixnet loopix-client <https://github.com/nymtech/nym-mixnet>.

Furthermore, when running the build binary, it requires two arguments: 1. name of the loopix-client and 2. port to run mixclient on.

For the first one, you need to run `init` command on the previously mentioned `loopix-client`. The port can be any valid port number.