#!/usr/bin/env bash
go build -v -tags=normal -o Gobot.out && strip Gobot.out && upx Gobot.out
