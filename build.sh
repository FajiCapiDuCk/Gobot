#!/bin/bash
go build -v -o Gobot.out && strip Gobot.out && upx Gobot.out
