registree
=========

A simple (and slow!) CLI tool to display relations between images stored in a private Docker registry as a set of trees.

[![Build Status](https://travis-ci.org/bjaglin/registree.svg?branch=master)](https://travis-ci.org/bjaglin/registree)

## Usage

Use the [automated build image](https://registry.hub.docker.com/u/bjaglin/registree/) to fetch data from your private registry (API v1) and dump the layer structure:

    docker run -e "REGISTRY_URL=http://myregistry.net" bjaglin/registree

Make sure you have the latest version via:

    docker pull bjaglin/registree

Note that the tool is only compatible with the Registry API v1. Debug logs can be enabled by setting the environment variable `REGISTREE_DEBUG=true`.

## Example

    f9d45d2aff66074564f921cf56084d0b1de379f894aaecbfd57a3c8030e5fa39 [] 221.4 MB
    1c966893d2a8e95d3b61e5f31680f73aa7752cda93ab6791473dc267bd64daea   [targeting:production_201506030921 targeting:production_2015007031600] 61.56 MB
    6b533c0982305dcee7f1348b0afe969b7013fee82436146a7ca4d7d0b1349e1e   [front-tools:2.1.12 front-tools:master npm-builder:2.1.12] 467 MB
    ce7b1346ab1a3d1cfc66e2cb15594c740fc385f0c5457761e3658162d65142cb   [] 255.8 MB
    2ea2a2f2ad2f0b4570d2557c506e8386deb3338d80ef4621b36dafcd54030708     [ssp:improvement_bid-request-dsl-call] 103.7 MB
    f6848dac6979b1682bf797f842450c1bc2912b24651bae4497d12f2e112f3623     [algo:improvement_bid-request-dsl-call] 63.42 MB
    ce98a4f8e6b59740d6bc82af2205c21435993135c5b5238c848177cc77ede6d8   [] 0 B
    f91becfbabfc5fc71f4edf928b048536ccdfc258adb0c256ef9affaa69f5d1a8     [cassandra:1.2.18] 91.05 MB
    46f23cc11fedc0d1334ac2ab91bb69ee3b2054736b77700a47cd9c4de992707d       [cassandra-migrations:feature_fixtures] 67.48 MB
    40d90bff69b052395e2d8be8ef6aa9529601150f3a4ce6c6169a6ae4276359e0     [tracking:refactor_trackservice] 83.18 MB

Indentation following the raw layer ids allows you to visualize the hierarchy between images (transformation to a DOT file is not implemented):

![Tree view](http://g.gravizo.com/g?
  digraph G {
    "2ea2a2f"[label="2ea2a2f\\nssp:impro..."];
    "f6848da"[label="f6848da\\nalgo:impro..."];
    "f91becf"[label="f91becf\\ncassandra:1.2.18"];
    "46f23cc"[label="46f23cc\\ncassandra-migrations:featur..."];
    "40d90bf"[label="40d90bf\\ntracking:refact..."];
    "1c96689"[label="1c96689\\ntargeting:prod...\\ntargeting:prod..."];
    "6b533c0"[label="6b533c0\\nfront-tools:2.1.12\\nfront-tools:master\\nnpm-builder:2.1.12"];
    "?" -> "f9d45d2" [label="221.4 MB"];
    "f9d45d2" -> "ce7b134" [label="255.8 MB"];
    "ce7b134" -> "2ea2a2f"[label="103.7 MB"];
    "ce7b134" -> "f6848da"[label="62.42 MB"];
    "f9d45d2" -> "ce98a4f" [label="0 B"];
    "ce98a4f" -> "f91becf" [label="91.05 MB"];
    "f91becf" -> "46f23cc" [label="67.48 MB"];
    "ce98a4f" -> "40d90bf" [label="83.18 MB"];
    "f9d45d2" -> "1c96689" [label="61.56 MB"];
    "f9d45d2" -> "6b533c0" [label="467 MB"];
  }
)
	
In this example, all images currently tagged in the repository inherit from the same base image (`f9d45d2`), which weights 221.4 MB (that includes the weigth of its potential ancestors not listed here), but does not have a tag in that repository (probably a base OS image coming from the central hub, but which was not pushed as such). `ssp:improvement_bid-request-dsl-call` and `algo:improvement_bid-request-dsl-call` closest common ancestor is `ce7b134`, and the respective weigths they bring-in on top of that image are 103.7 MB and 63.42 MB. `targeting:production_201506030921` and `targeting:production_2015007031600` are actually the same image, which has only `f9d45d2` (and its potential ancestors ommited here) in common with all other tagged images in the repository.
