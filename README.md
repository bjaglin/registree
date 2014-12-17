registree
=========

A simple CLI tool to display relations between images stored in a private Docker registry as a set of trees.

[![Build Status](https://travis-ci.org/bjaglin/registree.svg?branch=master)](https://travis-ci.org/bjaglin/registree)

## Usage

Use the [automated build image](https://registry.hub.docker.com/u/bjaglin/registree/) to get started:

    docker run -e "REGISTRY_URL=http://myregistry.net" bjaglin/registree

Make sure you have the latest version via:

    docker pull bjaglin/registree
