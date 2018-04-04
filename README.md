# react-sdk-config-server

[![#general:homeserver.today](https://img.shields.io/badge/matrix-%23general:homeserver.today-brightgreen.svg)](https://matrix.to/#/#general:homeserver.today)
[![TravisCI badge](https://travis-ci.org/homeserver-today/react-sdk-config-server.svg?branch=master)](https://travis-ci.org/homeserver-today/react-sdk-config-server)

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

The primary route at `/config(.*).json` is unauthenticated and calculates a configuration based on the domain name. The
domain name can either be specified in the config file (eg: `config.t2bot.io.json`) or via the `Host` header (the default
thing that happens when accessing `/config.json`).

The configuration is then calculated based on the various wildcard domains that are configured. Wildcard configs can be
set up by using the normal API routes and specifying `*` in the domain name. For example, `*.t2bot.io` can be used as a
domain name to match any subdomain of t2bot.io. This wildcard can be placed anywhere in the domain name, including on it's
own to provide a default config for all domains.

The order the templated (wildcard) configs are used is defined by a `hstoday.weight` key in the template. The templates
are used in ascending order by weight (therefore higher numbers 'win' if there's a conflict for which value to set a key
to). Templates without weights will be treated as weight 0. The domain's non-templated config will always be the highest
weight. If multiple templates share the same weight, the order is not defined.

### Getting a domain's configuration

This is the same as calling `/config.domain.json`, but provided for symmetry with the rest of the API.

**Example**:
```
$ curl -X GET -H "Authorization: Bearer TheSecretFromYourConfig" http://localhost:8000/api/v1/config/t2bot.io
{
    "brand": "Riot",
    "default_hs_url": "https://t2bot.io",
    "default_is_url": "https://vector.im"
}
```

### Setting a domain's configuration

It is recommended to first `GET` the config before trying to `PUT` a new one as this will replace the current value. Upon
a successful call, this will echo back the new config as a response.

**Example**:
```
$ curl -X PUT -H "Authorization: Bearer TheSecretFromYourConfig" -H "Content-Type: application/json" --data '{"brand":"Riot"}' http://localhost:8000/api/v1/config/t2bot.io
{
    "brand": "Riot"
}
```

**Example** (setting a weight of 12 to a wildcard domain):
```
$ curl -X PUT -H "Authorization: Bearer TheSecretFromYourConfig" -H "Content-Type: application/json" --data '{"brand":"Riot", "hstoday.weight": 12}' http://localhost:8000/api/v1/config/*.t2bot.io
{
    "brand": "Riot",
    "hstoday.weight": 12
}
```

### Deleting a domain's configuration

Any configuration may be deleted. An empty object is returned as a response to
signify that the configuration was deleted.

**Example**:
```
$ curl -X DELETE -H "Authorization: Bearer TheSecretFromYourConfig" http://localhost:8000/api/v1/config/t2bot.io
{}
```
