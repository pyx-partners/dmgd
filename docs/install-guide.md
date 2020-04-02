# Prova Install

## Install Go

1. Follow installation instructions here: https://golang.org/doc/install
2. Confirm go works with the "hello world" example in the install

## Check Go Version

Check that the version is correct and the GOROOT and the GOPATH are defined and not the same path. Make you have setup the go directory with bin/pkg/src sub directories for the source and binary builds.

```bash
$ go version
$ go env GOROOT GOPATH
```

GOPATH should be in your user directory so that there will be no issue writing there. An example would be `~/dev/go`

You should add a line to your PATH environment variable to include the GOPATH binary builds path.

```
export PATH=$PATH:$GOPATH/bin
```

## Clone Prova

Make sure you have an SSH key registered with GitHub, then create in your `GOPATH` src/ directory the path `github.com/bitgo` and navigate to this path. There may be errors along this path, see below for how to resolve.

```bash
$ go get -u github.com/Masterminds/glide
$ git clone git@github.com:BitGo/prova.git
$ glide install
$ go install
```

When executing `go install` to install Prova, make sure that you are in the GOPATH directory (github.com/pyx-partners/dmgd) otherwise it will give you an error instructing you to install from within your GOPATH directory.

After you install you should be able to do `go test`, `go install` and this will successfully test the top level of prova and build it into the go/bin. Because the bin path is in your PATH you should be able to do `prova -h` successfully to confirm the app has been built and is in your path.

## Run Prova

To run Prova you will need a data directory to store the chain state. Create this directory where you like, like for testnet in your home directory you can create ~/testnet/ - in this directory make a directory like `/1/` if you want to work with multiple chain states.

For Prova in regtest simulating 2 nodes, create directories `~/regtest/1` and `~/regtest/2` and the following test.sh bash script can run two nodes against each other in regtest.

```
#!/bin/bash
LOCAL=127.0.0.1
nohup prova --listen=$LOCAL:6001 --rpcuser=user --rpcpass=pass --rpclisten=$LOCAL:7001 --datadir=$HOME/regtest/1/ --connect=$LOCAL:6002 --regtest --txindex --miningaddr=TCq7ZvyjTugZ3xDY8m1Mdgm95v4QmNuqYXYbutQgDgHtW > $HOME/regtest/1/prova.log 2>&1 &
nohup prova --listen=$LOCAL:6002 --rpcuser=user --rpcpass=pass --rpclisten=$LOCAL:7002 --datadir=$HOME/regtest/2/ --connect=$LOCAL:6001 --regtest > $HOME/regtest/2/prova.log 2>&1 &
sleep 1
provactl -u user -P pass -s $LOCAL:7001 setvalidatekeys "[\"d36c82406d3c77ebc342aaa16f24a985fbfe63c75e6fd2afeffa1ba69632d252\"]"
provactl -u user -P pass -s $LOCAL:7001 generate 101
```

For Prova in testnet mode, create a ~/testnet/1 directory, use the following command line arguments to start the testnet node and connect to the remote node.

```
prova
--addpeer=$PEER_ADDRESS
--addrindex
--datadir=$YOUR_PATH_TO_DATA_DIRECTORY
--listen=127.0.0.1:6001
--notls
--rpclisten=127.0.0.1:7001
--rpcpass=pass
--rpcuser=user
--testnet
--txindex
```

These arguments will enable the tx and addr indexes, specify testnet and the data directory, and allow commands to the localhost RPC with user/pass on port 7001. Mainnet is not currently supported, so make sure to toggle testnet.

## Run Tests

To test an individual package, in the directory of the package run the command `go test -v`

