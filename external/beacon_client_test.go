package external

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"math/big"
	"testing"
	"time"
)

const NodeReal = "https://eth2-beacon-mainnet.nodereal.io/v1/a24f28a0f2484effa9fea36b8e281272"
const NodeRealEth1 = "https://eth-mainnet.nodereal.io/v1/a24f28a0f2484effa9fea36b8e281272"
const quickNode = "https://docs-demo.quiknode.pro"

func TestGetBlockHeight(t *testing.T) {

	beaconClient, err := NewBeaconClient(NodeReal, time.Second*3)
	if err != nil {
		panic(err)
	}
	sideCars, err := beaconClient.GetBlob(context.Background(), 8762217)
	if err != nil {
		panic(err)
	}

	fmt.Println(sideCars[4].Blob)

	//for _, sc := range sideCars {
	//
	//	if i
	//
	//	fmt.Println(sc.Index)
	//	//consensus, _ := sc.SignedBeaconBlockHeader.Message.ToConsensus()
	//	//fmt.Println(consensus.String())
	//	//fmt.Println(hex.EncodeToString(consensus.BodyRoot))
	//	fmt.Println(sc.KzgProof)
	//	fmt.Println(sc.KzgCommitment)
	//	fmt.Println(sc.Blob[:300])
	//}
}

func TestGetEth1BlockTx(t *testing.T) {

	ethClient, err := ethclient.Dial(NodeRealEth1)
	if err != nil {
		panic(err)
	}
	block, err := ethClient.BlockByNumber(context.Background(), big.NewInt(19530505))
	if err != nil {
		panic(err)
		return
	}
	for _, tx := range block.Body().Transactions {
		if tx.Type() == 3 {
			fmt.Println("blob tx hash: " + hex.EncodeToString(tx.Hash().Bytes()))
			fmt.Println("----------------------------")
			for _, bs := range tx.BlobHashes() {
				fmt.Println("blob versioned hash: " + bs.String())
			}

			fmt.Printf("blob tx value:  %d\n", tx.Value().Uint64())
			fmt.Printf("blob tx to addr:  %s\n", tx.To().String())

			fmt.Printf("blob tx data %s\n", hex.EncodeToString(tx.Data()))

			fmt.Printf("blob sidecar is nil %v\n", tx.BlobTxSidecar() == nil)

			//for _, sc := range tx().BlobHashes() {
			//	fmt.Println("blob versioned hash: " + sc.String())
			//}
		}
	}
}

func TestGetBlock(t *testing.T) {
	// 8731617
	// 19530503
	beaconClient, err := NewBeaconClient(NodeReal, time.Second*3)
	if err != nil {
		panic(err)
	}

	resp, err := beaconClient.GetLatestBlock(context.Background())
	if err != nil {
		panic(err)
	}

	resp, err = beaconClient.GetBlock(context.Background(), 8762217)
	if err != nil {
		panic(err)
	}

	switch resp.Version {
	case version.String(version.Deneb):
		sbb := &structs.SignedBeaconBlockDeneb{Message: &structs.BeaconBlockDeneb{}}
		err = json.Unmarshal(resp.Data.Message, sbb.Message)
		if err != nil {
			panic(err)
		}
		sbb.Signature = resp.Data.Signature
		signedBeaconBlockDeneb, err := sbb.ToConsensus()
		if err != nil {
			panic(err)
		}
		fmt.Println("eth1data " + signedBeaconBlockDeneb.GetBlock().GetBody().GetEth1Data().String())
		fmt.Printf("eth block number %d\n", signedBeaconBlockDeneb.GetBlock().GetBody().GetExecutionPayload().GetBlockNumber())

		//for i, tx := range signedBeaconBlockDeneb.GetBlock().GetBody().GetExecutionPayload().GetTransactions() {
		//	txss := &types.Transaction{}
		//	err = txss.UnmarshalJSON(tx)
		//	if err != nil {
		//		fmt.Println(err.Error())
		//		return
		//	}
		//	fmt.Println(txss)
		//	fmt.Printf("tx %d is %s\n", i, hex.EncodeToString(tx))
		//}
	}
}
