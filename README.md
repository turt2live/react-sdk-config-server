# react-sdk-config-server

[![#general:homeserver.today](https://img.shields.io/badge/matrix-%23general:homeserver.today-brightgreen.svg)](https://matrix.to/#/#general:homeserver.today)
[![TravisCI badge](https://travis-ci.org/turt2live/matrix-media-repo.svg?branch=master)](https://travis-ci.org/turt2live/matrix-media-repo)

RESTful service for configuring the per-domain config for matrix-react-sdk (Riot)

# Installing

Assuming Go 1.9 is already installed on your PATH:
```bash
# Get it
git clone https://github.com/homeserver-today/react-sdk-config-server
cd react-sdk-config-server

# Set up the build tools
currentDir=$(pwd)
export GOPATH="$currentDir/vendor/src:$currentDir/vendor:$currentDir:"$GOPATH
go get github.com/constabulary/gb/...
export PATH=$PATH":$currentDir/vendor/bin:$currentDir/vendor/src/bin"

# Build it
gb vendor restore
gb build

# Configure it (edit config-server.yaml to meet your needs)
cp config.sample.yaml config-server.yaml

# Run it
bin/config_server
```

### Installing in Alpine Linux

The steps are almost the same as above. The only difference is that `gb build` will not work, so instead use the following lines:
```bash
go build -o bin/config_server ./src/github.com/homeserver-today/react-sdk-config-server/cmd/config_server/
```

# Deployment

This is intended to run behind a load balancer next to your client's install (Riot). A sample nginx configuration for this is:

```ini
server {
  listen 80;
  listen [::]:80;
  # ssl configuration not shown

  root /var/www/html;
  index index.html;

  location / {
      allow all;
      try_files $uri $uri/ =404;
  }

  # Redirect requests for the config.json to react-sdk-config-server
  location ~ /config(.*).json {
      proxy_read_timeout 60s;
      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $remote_addr;
      proxy_pass http://localhost:8000; # Point this towards the react-sdk-config-server
  }
}
```

# API

TODO
