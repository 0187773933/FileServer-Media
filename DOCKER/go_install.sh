#!/bin/bash
GO_VERSION="1.22.2" # has to be hardcoded , or you have to dynamically re-write Dockerfile
ARCH=$(dpkg --print-architecture)
case $ARCH in
	armhf)
		GOLANG_ARCH_NAME="linux-armv6l"
		ARCHIVE_NAME=$(echo "go$GO_VERSION.$GOLANG_ARCH_NAME.tar.gz")
		ARCHIVE_URL=$(echo "https://golang.org/dl/$ARCHIVE_NAME")
		;;
	arm64)
		GOLANG_ARCH_NAME="linux-arm64"
		ARCHIVE_NAME=$(echo "go$GO_VERSION.$GOLANG_ARCH_NAME.tar.gz")
		ARCHIVE_URL=$(echo "https://golang.org/dl/$ARCHIVE_NAME")
		;;
	amd64)
		GOLANG_ARCH_NAME="linux-amd64"
		ARCHIVE_NAME=$(echo "go$GO_VERSION.$GOLANG_ARCH_NAME.tar.gz")
		ARCHIVE_URL=$(echo "https://golang.org/dl/$ARCHIVE_NAME")
		;;
	*)
		echo "unknown arch"
		#GOLANG_ARCH_NAME="windows-amd64.zip"
		;;
esac
echo "Downloading $ARCHIVE_URL"
wget $ARCHIVE_URL --progress=bar -O go.tar.gz
echo "Extracting To : /usr/local/go"
mkdir -p ~/go