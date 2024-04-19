#!/bin/bash

env GOOS=freebsd GOARCH=amd64 go build -o pws-handler.bsd
