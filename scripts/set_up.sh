#!/usr/bin/env bash

basedir=$(cd `dirname $0`; pwd)
WORKSPACE=${basedir}
source ${WORKSPACE}/.env
echo ${WORKSPACE}

function prepare() {
    cd ${WORKSPACE}
    git clone https://github.com/bnb-chain/greenfield-cmd.git
    cd greenfield-cmd
    git checkout allowance
    make build
    cd build

    touch key.txt & echo ${PRIVATE_KEY} > key.txt
    touch password.txt & echo "pass" > password.txt
    ./gnfd-cmd --home ./ --passwordfile password.txt account import key.txt

    touch config.toml
    {
      echo rpcAddr = \""${GREENFIELD_RPC}"\"
      echo chainId = \""${GREENFIELD_CHAIN_ID}"\"
    } > config.toml
    echo "prepared binary"
}

function create_bucket() {
    prepare
    ./gnfd-cmd -c ./config.toml --home ./ --passwordfile password.txt bucket create --primarySP ${SP_ADDRESS} --chargedQuota 322122547200 gnfd://${BUCKET_NAME}
    sleep 5
    ./gnfd-cmd -c ./config.toml --home ./ bucket head gnfd://${BUCKET_NAME}
    echo "bucket created"
}

function grant() {
    prepare
    ./gnfd-cmd -c ./config.toml --home ./ --passwordfile password.txt policy put --grantee ${GRANTEE_BUNDLE_ACCOUNT} --actions createObj  grn:b::"$BUCKET_NAME"
    sleep 5
    ./gnfd-cmd -c ./config.toml --home ./ --passwordfile password.txt fee grant --grantee ${GRANTEE_BUNDLE_ACCOUNT} --allowance ${ALLOWANCE}
    echo "granted permission"
}

function all() {
    prepare
    ./gnfd-cmd -c ./config.toml --home ./ --passwordfile password.txt bucket create --primarySP ${SP_ADDRESS} --chargedQuota 53687091200 gnfd://${BUCKET_NAME}
    sleep 5
    ./gnfd-cmd -c ./config.toml --home ./ bucket head gnfd://${BUCKET_NAME}
    ./gnfd-cmd -c ./config.toml --home ./ --passwordfile password.txt policy put --grantee ${GRANTEE_BUNDLE_ACCOUNT} --actions createObj  grn:b::"$BUCKET_NAME"
    sleep 5
    ./gnfd-cmd -c ./config.toml --home ./ --passwordfile password.txt fee grant --grantee ${GRANTEE_BUNDLE_ACCOUNT} --allowance ${ALLOWANCE}
    echo "created bucket and granted permission"
}

CMD=$1
case ${CMD} in
--create_bucket)
  create_bucket
  ;;
--grant)
  grant
  ;;
--all)
  all
  ;;
esac


