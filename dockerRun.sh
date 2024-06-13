#!/bin/bash
APP_NAME="public-file-server-media"
sudo docker rm -f $APP_NAME || echo ""
id=$(sudo docker run -dit \
--name $APP_NAME \
--restart="always" \
-v "$(pwd)"/SAVE_FILES:/home/morphs/SAVE_FILES \
-p 5754:5754 \
$APP_NAME /home/morphs/SAVE_FILES/config.yaml)
sudo docker logs -f $id