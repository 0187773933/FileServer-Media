#!/bin/bash

get_yaml_value() {
	local key_path=$1
	local yaml_file="/home/morphs/SAVE_FILES/config.yaml"
	local value=$(python3 -c "
import sys, yaml
with open('$yaml_file', 'r') as file:
	config = yaml.safe_load(file)
keys = '$key_path'.split('.')
value = config
try:
	for key in keys:
		value = value[key]
	print(value)
except KeyError:
	print('Key not found')
")
	echo "$value"
}

GIT_URL=$(get_yaml_value "git.url")
GOOS=$(get_yaml_value "go.os")
GOARCH=$(get_yaml_value "go.arch")
HASH_FILE="/home/morphs/git.hash"
REMOTE_HASH=$(git ls-remote "$GIT_URL" HEAD | awk '{print $1}')
BUILD_PATH="/home/morphs/DockerBuild"
EXEC_PATH="/home/morphs/DockerBuild/server"

if [ -f "$HASH_FILE" ]; then
	STORED_HASH=$(sudo cat "$HASH_FILE")
else
	STORED_HASH=""
fi

if [ "$REMOTE_HASH" == "$STORED_HASH" ]; then
	echo "No New Updates Available"
	cd "$BUILD_PATH"
	exec "$EXEC_PATH" "$@"
else
	echo "New updates available. Updating and Rebuilding Go Module"
	echo "$REMOTE_HASH" | sudo tee "$HASH_FILE"
	cd "/home/morphs"
	sudo rm -rf "$BUILD_PATH"
	git clone "$GIT_URL" "$BUILD_PATH"
	sudo chown -R morphs:morphs "$BUILD_PATH"
	cd "$BUILD_PATH"
	/usr/local/go/bin/go mod tidy
	GOOS="$GOOS" GOARCH="$GOARCH" /usr/local/go/bin/go build -o "$EXEC_PATH"
	exec "$EXEC_PATH" "$@"
fi