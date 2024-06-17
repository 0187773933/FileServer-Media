#!/bin/bash

# Set variables
LOCK_FILE="docker_build.lock"
HASH_FILE="git.hash"
CONTAINER_NAME="public-file-server"
GITHUB_REPO="https://github.com/0187773933/FileServer-Media"

# Check for a lock file
if [ -f "$LOCK_FILE" ]; then
    echo "Another build is in progress. Exiting."
    exit 1
fi

# Set a lock file
touch $LOCK_FILE

# Get the latest commit hash from the remote repository
REMOTE_HASH=$(git ls-remote https://github.com/0187773933/FileServer-Media.git HEAD | awk '{print $1}')

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
    sudo ./dockerRestart.sh
else
    echo "No updates available."
fi

# Remove the lock file
rm $LOCK_FILE