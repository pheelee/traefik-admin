# Traefik-Admin

![Go](https://github.com/pheelee/traefik-admin/workflows/Go/badge.svg)
![Go Version](https://img.shields.io/github/go-mod/go-version/pheelee/traefik-admin)
![Version](https://img.shields.io/github/v/tag/pheelee/traefik-admin?color=green&label=Version)


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