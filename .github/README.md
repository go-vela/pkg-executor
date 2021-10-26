# pkg-executor

> DISCLAIMER:
>
> This contents of this repository have been migrated into [go-vela/worker](https://github.com/go-vela/worker).
>
> This was done as a part of [go-vela/community#395](https://github.com/go-vela/community/issues/395) to deliver [on a proposal](https://github.com/go-vela/community/blob/master/proposals/2021/08-25_repo-structure.md).

[![license](https://img.shields.io/crates/l/gl.svg)](../LICENSE)
[![GoDoc](https://godoc.org/github.com/go-vela/pkg-executor?status.svg)](https://godoc.org/github.com/go-vela/pkg-executor)
[![Go Report Card](https://goreportcard.com/badge/go-vela/pkg-executor)](https://goreportcard.com/report/go-vela/pkg-executor)
[![codecov](https://codecov.io/gh/go-vela/pkg-executor/branch/master/graph/badge.svg)](https://codecov.io/gh/go-vela/pkg-executor)

Vela package designed for supporting the ability to run a [go-vela/worker](https://github.com/go-vela/worker) with different executors.

The following executors are supported:

* [Linux](https://www.linux.org/) - used to execute pipelines on a Linux distribution
* Local - used to execute pipelines with the Vela CLI

## Documentation

For installation and usage, please [visit our docs](https://go-vela.github.io/docs).

## Contributing

We are always welcome to new pull requests!

Please see our [contributing](CONTRIBUTING.md) docs for further instructions.

## Support

We are always here to help!

Please see our [support](SUPPORT.md) documentation for further instructions.

## Copyright and License

```
Copyright (c) 2021 Target Brands, Inc.
```

[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0)