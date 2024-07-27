#!/bin/bash

# so you have to run it once , without mounting the ADB_KEYS folder
# run some command , and pair with the fire cube
# then pull them locally :
	# sudo docker cp public-fire-c2-server:/home/morphs/.android ADB_KEYS
# and you are also going to want to set the local permissions
# you can run `id` inside the docker container to get uid and gid of morphs
# or set it in the docker file
# but should be :
	# sudo chown -R 1000:1000 ADB_KEYS/
	# sudo chown -R 1000:1000 SAVE_FILES/

# EVENT_DEVICE=$(readlink -f /dev/input/by-id/usb-Pulse-Eight_USB-CEC_Adapter_v7-if02-event-mouse)
#--device=$EVENT_DEVICE:/dev/input/event \
#MOUSE_DEVICE=$(readlink -f /dev/input/by-id/usb-Pulse-Eight_USB-CEC_Adapter_v7-if02-mouse)
#--privileged \

# sudo docker network create 6105-buttons

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

# APP_NAME=$(<../SAVE_FILES/Docker.name)
APP_NAME=$(get_yaml_value "docker.name")
APP_PORT=$(get_yaml_value "port")
sudo docker rm -f $APP_NAME || echo ""
id=$(sudo docker run -dit \
--name $APP_NAME \
--restart="always" \
-v $(pwd)/../SAVE_FILES/:/home/morphs/SAVE_FILES:rw \
-p $APP_PORT:$APP_PORT \
-e LOG_LEVEL=debug \
$APP_NAME /home/morphs/SAVE_FILES/config.yaml)
sudo docker logs -f $id

# --user=morphs \
# --privileged \
# --network=6105-buttons \
# --device=/dev/snd \
# --device=/dev/lirc0:/dev/lirc0 \
# --mount type=bind,source="$(pwd)"/SAVE_FILES/config.yaml,target=/home/morphs/config.yaml \