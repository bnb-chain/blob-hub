# Blob Syncer

## Overview

The Blob-Syncer service acts as an intermediary between Greenfield and Ethereum, continuously retrieving blobs from Ethereum.
These blobs, collected from a specific Beacon slot range, are consolidated into a bundle and transmitted to the [Bundle Service](https://docs.bnbchain.org/greenfield-docs/docs/api/bundle-service/)
for further processing. Subsequently, the bundle is stored in Greenfield for long-term archival purposes. Additionally,
the Blob-Syncer service offers APIs that enable users to access and retrieve historical blobs as needed.

## Disclaimer
**The software and related documentation are under active development, all subject to potential future change without
notification and not ready for production use. The code and security audit have not been fully completed and not ready
for any bug bounty. We advise you to be careful and experiment on the network at your own risk. Stay safe out there.**

## Components
- blob-syncer: This component is specifically engineered to synchronize blobs and securely store them in Greenfield. It includes a post-verification process to ensure the integrity of blob storage.
- blob-syncer-server: This component functions as the API server, catering to users' requests for querying blobs.

## Requirement

Go version above 1.20

## Prerequisite

### Create a bucket on Greenfield

if you don't have a bucket yet, set up one for blob storage. There are a few ways to create one, below shows examples via [greenfield-go-sdk](https://github.com/bnb-chain/greenfield-go-sdk)
and using provided script.

#### use go-sdk
```go
  account, err := types.NewAccountFromPrivateKey("test", privateKey)
	if err != nil {
		log.Fatalf("New account from private key error, %v", err)
	}
	cli, err := client.New("greenfield_5600-1", "https://gnfd-testnet-fullnode-tendermint-us.bnbchain.org:443", client.Option{DefaultAccount: account})
	if err != nil {
		log.Fatalf("unable to new greenfield client, %v", err)
	}
	ctx := context.Background()

	// get storage providers list
	spLists, err := cli.ListStorageProviders(ctx, true)
	if err != nil {
		log.Fatalf("fail to list in service sps")
	}
	// choose the first sp to be the primary SP
	primarySP := spLists[0].GetOperatorAddress()

	// create bucket
	_, err = cli.CreateBucket(ctx, bucketName, primarySP, types.CreateBucketOptions{})
	handleErr(err, "CreateBucket")
	log.Printf("create bucket %s on SP: %s successfully \n", bucketName, spLists[0].Endpoint)

	// head bucket
	bucketInfo, err := cli.HeadBucket(ctx, bucketName)
	handleErr(err, "HeadBucket")
	log.Println("bucket info:", bucketInfo.String())
```

#### use provided script
you can use the script, before runinng it, modify the the scripts/.env file(the GRANTEE_BUNDLE_ACCOUNT does not need to modified at this moment):

```shell
bash scripts/set_up.sh --create_bucket
```

### Get a Bundler Account

Request a bundle account from the Bundle Service, you need to grant the bundle account permission in next step, so that bundle service
can create object behave of your account.

```shell
curl -X POST  https://gnfd-testnet-bundle.nodereal.io/v1/bundlerAccount/0xf74d8897D8BeafDF4b766E19A62078DE84570656

{"address":"0x4605BFc98E0a5EA63D9D5a4a1Df549732a6963f3"}
```

### Grant fee and permission to the bundle address for creating bundled objects under the bucket


#### use go-sdk

```go
  bucketActions := []permTypes.ActionType{permTypes.ACTION_CREATE_OBJECT}
  statements := utils.NewStatement(bucketActions, permTypes.EFFECT_ALLOW, nil, sdktypes.NewStatementOptions{})
  bundleAgentPrincipal, err := utils.NewPrincipalWithAccount(sdk.MustAccAddressFromHex(bundleAddrToGrant))
  if err != nil {
  util.Logger.Fatalf("NewPrincipalWithAccount: %v", err)
  return
}

  _, err = cli.PutBucketPolicy(ctx, BlobBucketName, BundleAgentPrincipal, []*permTypes.Statement{&statements}, sdktypes.PutPolicyOption{})
  if err != nil {
  util.Logger.Fatalf("put policy failed: %v", err)
  return
}
```

grant allowance,

the example shows allowance amount of 1 BNB, considered each object creation gas consumed is 0.000006 BNB, it is approximately for 1666667 objects.

```go
  allowanceAmount := math.NewIntWithDecimal(1, 19)
  _, err = cli.GrantBasicAllowance(ctx, bundleAcct.String(), allowanceAmount, nil, gnfdsdktypes.TxOption{})
  if err != nil {
  util.Logger.Fatalf("grant fee allowance failed: %v", err)
  }

```

You can find similar example [permission](https://github.com/bnb-chain/greenfield-go-sdk/blob/master/examples/permission.go):

#### use provided sript
or you can use the script, replace `GRANTEE_BUNDLE_ACCOUNT` with the addr got from Bundle Service in step 2, and modify the `ALLOWANCE` to your expect amount, so that transacion
gas will be paid your account:

```shell
bash scripts/set_up.sh --grant
```

After above steps are done, you can start running the Blob Syncer Service.


## Build

### Build all

```shell
make build
```

### Build blob syncer

```shell
make build_syncer
```

### Build blob syncer api server

```shell
make build_server
```

## Run

### Run the Blob Syncer instance

```shell
./build/blob-syncer --config-path config/local/config-syncer.json
```

```json
{
  "bucket_name": "your-bucket",
  "start_slot": 8783000,
  "create_bundle_slot_interval": 10,
  "beacon_rpc_addrs": [
  "https://eth2-beacon-mainnet.nodereal.io"
  ],
  "bundle_service_endpoints": [
  "https://gnfd-testnet-bundle.nodereal.io"
  ],
  "eth_rpc_addrs": [
  "https://eth-mainnet.nodereal.io"
  ],
  "temp_dir": "temp",
  "private_key": "0x....",
  "db_config": {
    "dialect": "mysql",
    "username": "root",
    "password": "pass",
    "url": "/blob-syncer?charset=utf8&parseTime=True&loc=Local",
    "max_idle_conns": 10,
    "max_open_conns": 100
  },
  "metrics_config": {
    "enable": true,
    "http_address": ""
  },
  "log_config": {
    "level": "DEBUG",
    "filename": "",
    "max_file_size_in_mb": 0,
    "max_backups_of_log_files": 0,
    "max_age_to_retain_log_files_in_days": 0,
    "use_console_logger": true,
    "use_file_logger": false,
    "compress": false
  }
}
```

### Run the api server

```shell
./build/blob-syncer-server --config-path config/local/config-server.json --port 8080 
```

## Blob syncer API

The Blob syncer server provides eth compatible API to query historical blob


### Get blob sidecars.

* GET /eth/v1/beacon/blob_sidecars/{block_id}

| ParameterName | Type            | Description                                                                                                                                                          |
| ------------- | --------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| block_id      | string          | Block identifier. Can be one of: slot(beacon chain),  <hex encoded blockRoot with 0x prefix>. note: "head" (canonical head in node's view), "genesis", "finalized" are not support |
| indices       | array of string | Array of indices for blob sidecars to request for in the specified block. Returns all blob sidecars in the block if not specified                                    |


200: Ok response

```json
{
  "data": [
    {
      "index": "0",
      "blob": "0x00b900026b636f6e74656e745479706569696d6167652f706e6767636f6e7465006e745a0001c8aa1f8b0800000000000003c497e55714dc17ef67e8ee6e8691009666405a1c4a40a4bb4b4a40ba41a41ba44b910649e998a11ba4bbbb4b...",
      "kzg_commitment": "0x8f5b5ac395257c71080721a72dfbc2a4260184a9fe6442d53ab17cd3c7246cfc263fbad5f063456bcfefea2c2795378a",
      "kzg_proof": "0x9952be38421793ca564e3cb779e14345912184bd883b8532629c23e948ba5c29103ddd072d1fbbb5e521a9bee3ee7925",
      "kzg_commitment_inclusion_proof": [
        "0x82ba896ae27ae4d01108146fa4a8313522b966697b088ec0e8f1e53f56c83626",
        "0x256135c2cf896b0790ab66a3a9b5cbbe5971968cbc72fc151063e92f500440a2",
        "0x93de7d5c33984c7e6c91d486aa097662517e933655c2155c0857a05e43074da5",
        "0xc78009fdf07fc56a11f122370658a353aaa542ed63e44c4bc15ff4cd105ab33c",
        "0x536d98837f2dd165a55d5eeae91485954472d56f246df256bf3cae19352a123c",
        "0x9efde052aa15429fae05bad4d0b1d7c64da64d03d7a1854a588c2cb8430c0d30",
        "0xd88ddfeed400a8755596b21942c1497e114c302e6118290f91e6772976041fa1",
        "0x87eb0ddba57e35f6d286673802a4af5975e22506c7cf4c64bb6be5ee11527f2c",
        "0x26846476fd5fc54a5d43385167c95144f2643f533cc85bb9d16b782f8d7db193",
        "0x506d86582d252405b840018792cad2bf1259f1ef5aa5f887e13cb2f0094f51e1",
        "0xffff0ad7e659772f9534c195c815efc4014ef1e1daed4404c06385d11192e92b",
        "0x6cf04127db05441cd833107a52be852868890e4317e6a02ab47683aa75964220",
        "0x0600000000000000000000000000000000000000000000000000000000000000",
        "0x792930bbd5baac43bcc798ee49aa8185ef76bb3b44ba62b91d86ae569e4bb535",
        "0x16c5286816e0c2fe95421dc404efb8919aa762db0a15e852933a2ad965aa9ed5",
        "0xdb56114e00fdd4c1f85c892bf35ac9a89289aaecb1ebd0a96cde606a748b5d71",
        "0xd5be0ed2682550ce71db46a6d05873d67bbb855b43a9e017eb76afe317cf7e7d"
        ],
      "signed_block_header": {
        "message": {
          "body_root": "0xfeffb7e2e57b5dac8849ce45723c701033053788dd8615fd8e2ad68689ea2cbf",
          "parent_root": "0xd39e1b7b8c5c2226d80a071cf919744679b22d95ce241210e6dee5dd76317dce",
          "proposer_index": "452467",
          "slot": "8783262",
          "state_root": "0xf014944ead7b1524d3b3d3e76c0285e20ffb277f3778c5f6be63c487904204cf"
        }
      }
    }
  ]
} 
```