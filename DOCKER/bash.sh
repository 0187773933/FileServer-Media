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
APP_NAME=$(get_yaml_value "docker.name")
# get_yaml_value() {
# 	local key=$1
# 	grep "$key:" "../SAVE_FILES/config.yaml" | awk -F ': ' '{print $2}' | tr -d '"'
# }
# APP_NAME=$(<../SAVE_FILES/Docker.name)
# APP_NAME=$(get_yaml_value "docker_name")
sudo docker exec -it "$APP_NAME" bash