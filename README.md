# eccd

<img align="right" width="159px" src="https://i.imgur.com/01csa1d.png">

[![Build Status](https://travis-ci.com/EnsicoinDevs/eccd.svg?branch=master)](https://travis-ci.com/EnsicoinDevs/eccd)
[![Go Report Card](https://goreportcard.com/badge/github.com/EnsicoinDevs/eccd)](https://goreportcard.com/report/github.com/EnsicoinDevs/eccd)
[![Coverage Status](https://coveralls.io/repos/github/EnsicoinDevs/eccd/badge.svg?branch=master)](https://coveralls.io/github/EnsicoinDevs/eccd?branch=master)
[![GoDoc](https://godoc.org/github.com/EnsicoinDevs/eccd?status.svg)](https://godoc.org/github.com/EnsicoinDevs/eccd)

An implementation of the Ensicoin protocol developed in Go.

## Getting started

To install eccd, there are two methods. You can compile the latest version from this repository, or you can use the docker image.

### Manually

Eccd uses the Go Modules support built into Go 1.11 to build. The easiest is to clone eccd in a directory outside of `GOPATH`, as in the following example:

```zsh
cd $HOME/src
git clone https://github.com/EnsicoinDevs/eccd.git
cd eccd
go install
```

If all goes well, eccd should now be accessible in your terminal.

### Using Docker

You can also use Docker to launch an instance very quickly:

```zsh
docker run -p 4224:4224 -p 4225:4225 johynpapin/eccd
```
