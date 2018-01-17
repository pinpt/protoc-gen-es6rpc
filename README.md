<div align="center">
	<img width="500" src=".github/logo.svg" alt="pinpt-logo">
</div>

<p align="center" color="#6a737d">
	<strong>This repo contains a protocol buffer compiler for generating ES6 classes.</strong>
</p>

## Background

This repo contains an implementation of a protocol buffer compiler which generates ES6 classes to communicate with a backend RPC service defined in the same protocol buffer file.

## Usage Example

```
protoc --proto_path=$GOPATH/src:. --es6rpc_out=apiprefix=/foo/bar:build test.proto
```

## License

Copyright (c) 2018 by PinPT, Inc. All Rights Reserved. Licensed under the Apache Public License v2.
