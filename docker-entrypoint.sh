#!/bin/sh

CONFIG_PATH="${CONFIG_PATH:-"/etc/canary-ng.yaml"}"

canary-ng -version
canary-ng -config "$CONFIG_PATH"
