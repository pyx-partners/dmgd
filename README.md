DMG
===

[![Build Status](https://travis-ci.org/pyx-partners/dmgd.svg?branch=master)](https://travis-ci.org/pyx-partners/dmgd.svg?branch=master)

DMG is a distributed consensus system for digital asset tokens written in Go (golang).
DMG was built to support RMG, a digitized form of gold introduced by [The Royal Mint](http://www.royalmint.com/rmg)
and CME Group.  Features include:

 * Asset Issuance

   DMG is a simple, single asset blockchain with asset issuance rather than mining.

 * Multi-signature only

   All accounts are multi-signature wallets.

 * No “black holes”

   The system enforces rules making it impossible to send to invalid or non-recoverable addresses

 * Permissioned Participants

   DMG can enforce only authorized validators and accounts.

 * Trusted Technology

   DMG is based on mature blockchain technologies.

   The implementation is derived from [Prova](https://github.com/BitGo/prova) and [btcd](https://github.com/btcsuite/btcd).

It downloads, validates, and serves a block chain using consensus rules for block acceptance.  It includes a full block validation testing framework.

It relays newly generated blocks, maintains a transaction memory pool, and
relays individual transactions that have not yet made it into a block.  It
ensures all individual transactions admitted to the pool follow the rules
required by the block chain.

DMG does *NOT* include wallet functionality.  This means you can't actually make or receive payments directly with DMG.  That functionality is provided by DMG wallet implementations.

## Requirements

[Go](http://golang.org) 1.13 or newer.

## Installation

#### Run in Docker container

A DMG node can be built into a Docker container and run using:

```bash
git clone https://github.com/pyx-partners/dmgd dmgd
cd dmgd
docker build --no-cache -t dmgd .
docker run -d dmgd:latest
```
Modify the `dmgd.conf` file to configure DMG and use `-p` option to open up required ports for RPC access and peer-to-peer port access.

#### Linux/BSD/MacOSX/POSIX - Build from Source

- Install Go according to the installation instructions here:
  http://golang.org/doc/install

- Ensure Go was installed properly and is a supported version:

```bash
$ go version
$ go env GOROOT GOPATH
```

If `GOROOT` or `GOPATH` is not set properly, add the following to your ~/.bashrc or /.profile startup files (substitute with proper directories if you have a custom installation of Go):

```bash
$ export GOROOT=/usr/local/go
$ export GOPATH=$HOME/go
$ export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```

The 'go' binary will be located under GOROOT, while your saved packages will be located under GOPATH.

NOTE: The `GOROOT` and `GOPATH` above must not be the same path.  It is
recommended that `GOPATH` is set to a directory in your home directory such as
`~/go` to avoid write permission issues.  It is also recommended to add
`$GOPATH/bin` to your `PATH` at this point.

- Run the following commands to obtain DMG, all dependencies, and install it:

```bash
$ git clone https://github.com/pyx-partners/dmgd $GOPATH/src/github.com/pyx-partners/dmgd
$ cd $GOPATH/src/github.com/pyx-partners/dmgd
$ go install . ./cmd/...
```

- DMG (and utilities) will now be installed in ```$GOPATH/bin```.  If you did
  not already add the bin directory to your system path during Go installation,
  we recommend you do so now.

## Updating

#### Linux/BSD/MacOSX/POSIX - Build from Source

- Run the following commands to update DMG, all dependencies, and install it:

```bash
$ cd $GOPATH/src/github.com/pyx-partners/dmgd
$ git pull
$ GO111MODULE=on go install . ./cmd/...
```

## Getting Started

DMG has several configuration options avilable to tweak how it runs, but all
of the basic operations described in the intro section work with zero
configuration.

#### Linux/BSD/POSIX/Source

```bash
$ ./dmgd
```

## Issues

The [integrated github issue tracker](https://github.com/pyx-partners/dmgd/issues)
is used for this project.

When reporting security issues, responsible disclosure is encouraged. The DMG developers at Pyx-Partners should be directly contacted at security@pyxpartners.com

## Documentation

The documentation is a work-in-progress.  It is located in the [docs](https://github.com/pyx-partners/dmgd/tree/master/docs) folder.

## License

DMG is licensed under the [copyfree](http://copyfree.org) ISC License.
