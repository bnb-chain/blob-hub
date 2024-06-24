# Blob Hub

## Overview

The blob-hub service acts as an intermediary between Greenfield and Ethereum/BSC, continuously retrieving blobs from Ethereum/BSC and persisting in Greenfield..
These blobs, collected from a specific Beacon slot range on Ethereum or specific block height range on BSC, are consolidated into a bundle and transmitted to the [Bundle Service](https://docs.bnbchain.org/greenfield-docs/docs/api/bundle-service/)
for further processing. Subsequently, the bundle is stored in Greenfield for long-term archival purposes. Additionally,
the blob-hub service offers APIs that enable users to access and retrieve historical blobs as needed.

## Disclaimer
- Data Accessibility. Due to the limited size of the Greenfield traffic package purchased, this service does not guarantee constant data access but ensures eventual access.
- Data latency. As the blob syncer need to bundle blobs across multi blocks, the latency is around 10-20 minutes before the blob is accessible on Greenfield.

## Components
- **syncer**: This component is specifically engineered to synchronize blobs and securely store them in Greenfield. It includes a post-verification process to ensure the integrity of blob storage.
- **server**: This component functions as the API server, catering to users' requests for querying blobs.

## Support Network
| Network  | Endpoint    | Bucket      |
|----------|-------------|-------------|
| Ethereum | https://gnfd-blobhub.bnbchain.org | [ethereum-mainnet-blobs](https://greenfieldscan.com/bucket/0x00000000000000000000000000000000000000000000000000000000000019e7) |
| BSC      | coming soon | coming soon |

Besides the mainnet setting up, we provide a Greenfield testnet(where data could be lost) endpoints:

| Network  | Endpoint                                  | Bucket                                                                                                                        |
|----------|-------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------|
| Ethereum | https://gnfd-blobhub-testnet.bnbchain.org | [ethereum-blob](https://testnet.greenfieldscan.com/bucket/0x0000000000000000000000000000000000000000000000000000000000003b57) |



## Blob Syncer API

The Blob syncer server provides eth compatible API to query historical blob

### Get blob sidecars.

* GET /eth/v1/beacon/blob_sidecars/{block_id}?indices={indices}

| ParameterName | Type            | Description                                                                                                                                                                        |
|---------------|-----------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| block_id      | string          | Block identifier. Can be one of: slot(beacon chain),  <hex encoded blockRoot with 0x prefix>. note: "head" (canonical head in node's view), "genesis", "finalized" are not support |
| indices       | array of string | Array of indices for blob sidecars to request for in the specified block. Returns all blob sidecars in the block if not specified, like "indices=0,1                               |


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

## Access to Blob Data Directly in Greenfield

In scenarios where the Bundle Service is inaccessible, direct access to blob data stored in Greenfield may be necessary, as blobs are consolidated into a bundle object.
User can retrieve the bundle object and extract specific blobs from it.

### Namimg pattern

The bundle objects uploaded to Greenfield adhere to the naming pattern `blobs_s{startSlot}_e{endSlot}`. For instance, if a bundle encompasses blobs within 30 slots, the bundle object could be named `blobs_s8864048_e8864077`, where 8864048 represents the start slot and 8864077 denotes the end slot. The individual blobs contained within this bundle follow the naming convention `blob_h{slot}_i{index}`, such as `blob_h8864074_i3`.

### Retrieving Blobs via Bundle SDK
The [bundle SDK](https://github.com/bnb-chain/greenfield-bundle-sdk) offers commands to fetch the bundle object from Greenfield. Follow the steps below:


```bash
git submodule update --init --recursive
cd bundle-sdk && make build

./build/bundler download -bucket ethereum-blob -object blobs_s8864048_e8864077 -chain-id greenfield_5600-1 -rpc-url https://gnfd-testnet-fullnode-tendermint-us.bnbchain.org:443  -output ./tmp
```

Once the bundle is downloaded and extracted, all original blob files can be found within the `tmp` directory.


## Setting Up Blob Syncer

### Requirement

Go version above 1.22

### Create a bucket on Greenfield

if you don't have a bucket yet, set up one for blob storage. There are a few ways to create one, below shows examples via [greenfield-go-sdk](https://github.com/bnb-chain/greenfield-go-sdk)
and using provided script.

#### Use Dcellar
Visit https://dcellar.io/ and create a bucket with at least 100G read quota per month.

#### use provided script
You can use the script, before runinng it, modify the the scripts/.env file(the GRANTEE_BUNDLE_ACCOUNT does not need to modified at this moment):

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

#### Use provided sript
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

### Build Blob Hub api server

```shell
make build_server
```

## Run

### Run the Blob Syncer instance

```shell
./build/syncer --config-path config/local/config-syncer.json
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
    "url": "/blob-hub?charset=utf8&parseTime=True&loc=Local",
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
./build/server --config-path config/local/config-server.json --port 8080
```