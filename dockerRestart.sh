#!/bin/bash
APP_NAME="public-file-server-media"
id=$(sudo docker restart $APP_NAME)
sudo docker logs -f $id