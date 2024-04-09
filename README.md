run the syncer instance:
```shell
./build/blob-syncer --config-type local --config-path config/config.json
````

run the server instance

```shell
./build/blob-syncer-server --config-path config/config.json --port 8080 
```

# Blob syncer

## Overview

The blob-syncer is the service sitting between Greenfield and Ethereum, constantly fetching blobs from Ethereum and persisting 
them into Greenfield storage for permanent archiving. The blob-syncer service also provides APIs for users to query historical blobs.

## Components
- syncer: the syncer is designed to sync blobs and store into Greenfield.
- server: the server is to serve user query request.

## Build

### Build blob syncer

```shell
make build_syncer
```

### Build blob syncer api server

```shell
make build_server
```

## Run

### Run blob syncer instance

```shell
./build/blob-syncer --config-path config/local/config-syncer.json
```

### Run the api server

```shell
./build/blob-syncer-server --config-path config/local/config-server.json --port 8080 
```

## Blob syncer API

The Blob syncer server provides eth compatible API to query historical blob


### Get blob sidecars.

* GET /eth/v1/beacon/blob_sidecars/{block_id}

| ParameterName | Type   | Description                                                                                                                                                            |
|--------------| ------ |------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| block_id     | string | Block identifier. Can be one of:  <slot>, <hex encoded blockRoot with 0x prefix>. note: "head" (canonical head in node's view), "genesis", "finalized" are not support |
| indices         | array<string> | Array of indices for blob sidecars to request for in the specified block. Returns all blob sidecars in the block if not specified                                      |


200: Ok response

```json
{
  "data": [
    {
      "index": "1",
      "blob": "0x0006eac4e2fac2ca844810be0dc9e398fa4961656c022b65e4af13728152980a00ed0800010017da0001003eff00010039d6000100bedd0001004ffb0001000000e600010022e000010004e5000000014df70a66ff064061f10bff8211f2ff8000c0040e8301a7f7941096..",
      "kzg_commitment": "0xb94c3ea2014aa34424b0e619ae91cb2c7299f219160c0702e7c0c66f063b7d368e1d881dd35f8b53ae00490abc455c64",
      "kzg_proof": "0xb542bb5619c8609c260133c0b0fb90e889aa2080d44ea842de26de30280acbbdf4c197a4f3620575cb6fa3ff614fec2e",
      "commitmentInclusionProof": [
        "0xeb4561cc212f3ff20b46b842abe512ef4fe80e4f7ba19a157e990973317e576d",
        "0xdd5a6c8cfbc72ee48dc5d5b8d912d09bd70de2ec11ae0a57d9a2e8531d9d561c",
        "0xc59bf2f2ec6443c468fc918dd7ddec3b588a899ee8b165866cf43898311b18fa",
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
        "0x810ee2cb21cca3e51a02375a8008f1aacd0a26920b83fdc1f822b09535fe7364",
        "0xdb56114e00fdd4c1f85c892bf35ac9a89289aaecb1ebd0a96cde606a748b5d71",
        "0x1047617013a89e7f8d63f77c0c8d18d823e9d27770697cf4aedf85bd381c25f5"
      ],
      "signedBeaconBlockHeader": {
        "message": {
          "bodyRoot": "5d07afb7c733c581dde806d3f28558db1a5fb8e79aaea551e5bc256da31dbd68",
          "parentRoot": "0feeeb3c6690b376c58d1aa9bf5ef756617969eaeec6862331cb740c1e4c69b4",
          "proposerIndex": "805060",
          "slot": "8777509",
          "stateRoot": "47b2a6ee35da2c4605d4d184b3cac2b961e67dc8763bf27085ac337c1d4018f2"
        }
      }
    }
  ]
} 
```
