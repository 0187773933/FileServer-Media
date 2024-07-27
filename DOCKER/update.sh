#!/bin/bash

get_yaml_value() {
	local key_path=$1
	local yaml_file="../SAVE_FILES/config.yaml"
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

# Set variables
LOCK_FILE="docker_build.lock"
HASH_FILE="git.hash"
GIT_REPO=$(get_yaml_value "git.url")

# Check for a lock file
if [ -f "$LOCK_FILE" ]; then
	echo "Another build is in progress. Exiting."
	exit 1
fi

# Set a lock file
touch $LOCK_FILE

# Get the latest commit hash from the remote repository
REMOTE_HASH=$(git ls-remote "$GIT_REPO" HEAD | awk '{print $1}')

# Check if the hash file exists and read the last stored hash
if [ -f "$HASH_FILE" ]; then
	STORED_HASH=$(cat "$HASH_FILE")
else
	STORED_HASH=""
fi

# Compare the hashes
if [ "$REMOTE_HASH" != "$STORED_HASH" ]; then
	echo "New updates available. Updating and restarting the container."
	echo "$REMOTE_HASH" > "$HASH_FILE"
	sudo ./restart.sh
else
	echo "No updates available."
fi

# Remove the lock file
rm $LOCK_FILE