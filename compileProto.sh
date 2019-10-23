#!/usr/bin/env bash

protoc --go_out=. ./message/message.proto
# cd message && protoc --js_out=import_style=commonjs,binary:../src ./message.proto
