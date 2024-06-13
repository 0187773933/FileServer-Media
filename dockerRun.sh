#!/bin/bash
APP_NAME="public-file-server-media"
sudo docker rm -f $APP_NAME || echo ""
id=$(sudo docker run -dit \
--name $APP_NAME \
--restart="always" \
--mount type=bind,source="$(pwd)"/config.yaml,target=/home/morphs/config.yaml \
-p 5754:5754 \
$APP_NAME /home/morphs/config.yaml)
sudo docker logs -f $id