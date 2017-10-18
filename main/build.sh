#!/bin/bash
export BASEDIR=`pwd`

export GOPATH=${BASEDIR}"/../gopath"

go clean
go version
time go build -v -o ../bin/router
echo 'finished'
