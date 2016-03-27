#!/bin/bash

set -e

pushd $GOPATH/src/github.com/kopwei/goovs/ > /dev/null
echo "Restoring dependencies"
godep restore
popd > /dev/null 
