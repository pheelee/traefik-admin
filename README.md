# Traefik-Admin

![Go Version](https://img.shields.io/github/go-mod/go-version/pheelee/traefik-admin)
![Version](https://img.shields.io/github/v/tag/pheelee/traefik-admin?color=green&label=Version)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/pheelee/traefik-admin/total)


This is a small webinterface to manage dynamic configs for traefik.

## Build on Homeassistant for testing
The following section describes the build process for the Home Assistant Operating System installation. For other installation methods please adopt accordingly.

Clone this repo to the `/addons` folder on your homeassistant instance (e.g using the SSH & Web Terminal Addon). Then edit the `config.json` and remove the line

`"image": "pheelee/hassio-addon-traefik-{arch}",`

In the addon store click the three dots (upper right corner) and select **Reload**. After that the addon should be visible in the **Local add-ons** section and be installable like any other addon.

## Build and publish to registry locally

```bash
docker run --rm --privileged -v ~/.docker:/root/.docker -v `pwd`:/data -v /var/run/docker.sock:/var/run/docker.sock homeassistant/amd64-builder --all -t /data
```
for more information about the homeassistant builder please visit https://github.com/home-assistant/builder

## Build using Github Actions
see `.github/workflows/release.yml` in this repo for inspiration

# MIT License
Copyright 2021-2024 Philipp Ritter

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.