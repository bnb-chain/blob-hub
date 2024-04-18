#!/usr/bin/env bash

basedir=$(cd `dirname $0`; pwd)
WORKSPACE=${basedir}
echo ${WORKSPACE}

cd ${WORKSPACE}
git clone https://github.com/bnb-chain/greenfield-cmd.git
cd greenfield-cmd
git checkout master
make build
cd build

./gnfd-cmd account new

./gnfd-cmd account export --unarmoredHex --unsafe
