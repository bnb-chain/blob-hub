package external

import (
	"context"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"

	"github.com/bnb-chain/blob-hub/config"
	"github.com/bnb-chain/blob-hub/external/eth"
	types2 "github.com/bnb-chain/blob-hub/types"
	"github.com/bnb-chain/blob-hub/util"
)

const BSCBlockConfirmNum = 3

type IClient interface {
	GetBlob(ctx context.Context, blockID uint64) ([]*types2.GeneralSideCar, error)
	GetBlockHeader(ctx context.Context, height uint64) (*types.Header, error)
	GetFinalizedBlockNum(ctx context.Context) (uint64, error)
	BlockByNumber(ctx context.Context, int2 *big.Int) (*types.Block, error)

	// for eth beacon chain
	GetLatestBeaconBlock(ctx context.Context) (*structs.GetBlockV2Response, error)
	GetBeaconHeader(ctx context.Context, slotNumber uint64) (*structs.GetBlockHeaderResponse, error)
	GetBeaconBlock(ctx context.Context, slotNumber uint64) (*structs.GetBlockV2Response, error)
}

type Client struct {
	ethClient    *ethclient.Client
	rpcClient    *rpc.Client
	beaconClient *eth.BeaconClient
	cfg          *config.SyncerConfig
}

func NewClient(cfg *config.SyncerConfig) IClient {
	ethClient, err := ethclient.Dial(cfg.RPCAddrs[0])
	if err != nil {
		panic("new eth client error")
	}
	cli := &Client{
		cfg:       cfg,
		ethClient: ethClient,
	}
	if cfg.Chain == config.BSC {
		rpcClient, err := rpc.DialContext(context.Background(), cfg.RPCAddrs[0])
		if err != nil {
			panic("new rpc client error")
		}
		cli.rpcClient = rpcClient
	} else {
		beaconClient, err := eth.NewBeaconClient(cfg.BeaconRPCAddrs[0])
		if err != nil {
			panic("new eth client error")
		}
		cli.beaconClient = beaconClient
	}
	return cli
}

func (c *Client) GetBlob(ctx context.Context, blockID uint64) ([]*types2.GeneralSideCar, error) {
	sidecars := make([]*types2.GeneralSideCar, 0)
	if c.cfg.Chain == config.BSC {
		var txSidecars []*BSCBlobTxSidecar
		number := rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(blockID))
		err := c.rpcClient.CallContext(ctx, &txSidecars, "eth_getBlobSidecars", number.String())
		if err != nil {
			return nil, err
		}
		if txSidecars == nil {
			return nil, ethereum.NotFound
		}
		idx := 0
		for _, txSidecar := range txSidecars {
			txIndex, err := util.HexToUint64(txSidecar.TxIndex)
			if err != nil {
				return nil, err
			}
			for j := range txSidecar.BlobSidecar.Blobs {
				sidecars = append(sidecars,
					&types2.GeneralSideCar{
						Sidecar: structs.Sidecar{
							Index:         strconv.Itoa(idx),
							Blob:          txSidecar.BlobSidecar.Blobs[j],
							KzgCommitment: txSidecar.BlobSidecar.Commitments[j],
							KzgProof:      txSidecar.BlobSidecar.Proofs[j],
						},
						TxIndex: int64(txIndex),
						TxHash:  txSidecar.TxHash,
					},
				)
				idx++
			}
		}
		return sidecars, err
	}
	ethSidecars, err := c.beaconClient.GetBlob(ctx, blockID)
	if err != nil {
		return nil, err
	}
	for _, sidecar := range ethSidecars {
		sidecars = append(sidecars,
			&types2.GeneralSideCar{
				Sidecar: *sidecar,
			},
		)
	}
	return sidecars, nil
}

func (c *Client) GetBlockHeader(ctx context.Context, height uint64) (*types.Header, error) {
	header, err := c.ethClient.HeaderByNumber(ctx, big.NewInt(int64(height)))
	if err != nil {
		return nil, err
	}
	return header, nil
}

func (c *Client) GetFinalizedBlockNum(ctx context.Context) (uint64, error) {
	var head *types.Header
	if err := c.rpcClient.CallContext(ctx, &head, "eth_getFinalizedHeader", BSCBlockConfirmNum); err != nil {
		return 0, err
	}
	if head == nil || head.Number == nil {
		return 0, ethereum.NotFound
	}
	return head.Number.Uint64(), nil
}

func (c *Client) BlockByNumber(ctx context.Context, int2 *big.Int) (*types.Block, error) {
	return c.ethClient.BlockByNumber(ctx, int2)
}

func (c *Client) GetLatestBeaconBlock(ctx context.Context) (*structs.GetBlockV2Response, error) {
	return c.beaconClient.GetLatestBeaconBlock(ctx)
}

func (c *Client) GetBeaconHeader(ctx context.Context, slotNumber uint64) (*structs.GetBlockHeaderResponse, error) {
	return c.beaconClient.GetBeaconHeader(ctx, slotNumber)
}

func (c *Client) GetBeaconBlock(ctx context.Context, slotNumber uint64) (*structs.GetBlockV2Response, error) {
	return c.beaconClient.GetBeaconBlock(ctx, slotNumber)
}

// BSCBlobSidecar is a sidecar struct for BSC
type BSCBlobSidecar struct {
	Blobs       []string `json:"blobs"`
	Commitments []string `json:"commitments"`
	Proofs      []string `json:"proofs"`
}

// BSCBlobTxSidecar is a sidecar struct for BSC blob tx
type BSCBlobTxSidecar struct {
	BlobSidecar BSCBlobSidecar `json:"blobSidecar"`
	BlockNumber string         `json:"blockNumber"`
	BlockHash   string         `json:"blockHash"`
	TxIndex     string         `json:"txIndex"`
	TxHash      string         `json:"txHash"`
}
