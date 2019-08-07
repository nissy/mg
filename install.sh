#!/bin/sh
RELEASES_LATEST_VERSION=`curl -fsSLI -o /dev/null -w %{url_effective} https://github.com/nissy/mg/releases/latest | sed 's/https\:\/\/github.com\/nissy\/mg\/releases\/tag\///'`
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    OS=$(uname | tr '[:upper:]' '[:lower:]')
    if [ "$OS" = "darwin" ]; then
        RELEASES_LATEST_VERSION_URL="https://github.com/nissy/mg/releases/download/${RELEASES_LATEST_VERSION}/mg-${RELEASES_LATEST_VERSION}_darwin_amd64.tar.gz"
    elif [ "$OS" = "linux" ]; then
        RELEASES_LATEST_VERSION_URL="https://github.com/nissy/mg/releases/download/${RELEASES_LATEST_VERSION}/mg-${RELEASES_LATEST_VERSION}_linux_amd64.tar.gz"
    fi
fi
if [ -n "$RELEASES_LATEST_VERSION_URL" ]; then
    echo ${RELEASES_LATEST_VERSION_URL}
    curl -L -O ${RELEASES_LATEST_VERSION_URL} | tar zxvf
fi
