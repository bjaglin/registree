registree
=========

A simple CLI tool to display relations between images stored in a private Docker registry as a set of trees.

[![Build Status](https://travis-ci.org/bjaglin/registree.svg?branch=master)](https://travis-ci.org/bjaglin/registree)

## Usage

Use the [automated build image](https://registry.hub.docker.com/u/bjaglin/registree/) to fetch data from your private registry (API v1) and dump the layer structure:

    docker run -e "REGISTRY_URL=http://myregistry.net" bjaglin/registree

Make sure you have the latest version via:

    docker pull bjaglin/registree

Note that the tool is only compatible with the Registry API v1.

## Output

Each line represents a layer: the fixed-length prefix is the layer id, followed by some spacing allowing to distinguish siblings from ancestors (roots are only only separated by a single space), and the potential tag(s) of that layer. Only layers that have either a tag or more than one child are listed, so that the trees appear less deep than they actually are, while still displaying relations between them. Note that base layers such as OS images that are available from the central registry will not show up in the output unless they have been explicitly tagged within the target registry.

From the following example, we can tell for instance that `oraclejre:1.8.0_11` is an ancestor of `selenium:2.42.2` and that no layer between them was tagged, that `0c9f511ac247` was tagged twice, and that `sbtdist-runner:java1.8.0_11` and `sbtnative-runner:java1.8.0_11` share a common ancestor layer which is not tagged in that registry.

    05489018501eba827a37365bdd74c687b6760846f51a5943f30e5cd4e0b6b6f2 [oraclejre:1.8.0_11]
    bafa304de54d84d0feb0d44401d6fb14cf7b710a2068b697ce9dd5b9e520f90f   [selenium:2.42.2]
    148a58f58eee25156c51562bf139f712dceee2cd2b7458e44249fc915f9c9bf4     [selenium-hub:2.42.2]
    68227fc7161bc2db50223e3d8f0360239dff321d59782d3516df2c2a7cdccbbc   [selenium:2.44.0 selenium:master]
    0c9f511ac247d1f42ee4d561179f7ddbe332aee4b921ec769b8f4b2094b18c24     [selenium-node-chrome:2.44.0_chrome39.0 selenium-node-chrome:master]
    57b4f9a70dd5e53fa4870b9e1442d07ae569de45d5eee2d9e5ea65d0327e5286     [selenium-hub:2.44.0 selenium-hub:master]
    cba62eeae5ca6e20aaa87ea3c75dc947e472b3fdc18d6db262d5037c86891d09   []
    a9335cff89e14ee51ba5dafddf33d117a614d2d9092f71c2bf7e7e674c39e855     [sbtdist-runner:java1.8.0_11]
    1d350bd93ba5a1035b35ddc672bd8386b9e9b2b8a1a4cc16205c6e5200515c3e     [sbtnative-runner:java1.8.0_11]
