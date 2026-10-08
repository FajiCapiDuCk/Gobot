#!/usr/bin/env bash
go build -tags=VPS -v -o Gobot.out && strip Gobot.out && upx Gobot.out
