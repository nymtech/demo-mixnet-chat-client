OUTDIR=build

all:
	make build_client

build_client:
	mkdir -p build
	go build -o $(OUTDIR)/chat-client ./cmd/chat-client
