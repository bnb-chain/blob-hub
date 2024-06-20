package external

import (
	"context"
	"math/big"
	"strconv"

	"github.com/bnb-chain/blob-hub/config"
	"github.com/bnb-chain/blob-hub/external/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
)

const BSCBlockConfirmNum = 3

type IClient interface {
	GetBlob(ctx context.Context, blockID uint64) ([]*structs.Sidecar, error)
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

func (c *Client) GetBlob(ctx context.Context, blockID uint64) ([]*structs.Sidecar, error) {
	if c.cfg.Chain == config.BSC {
		var r []*BlobTxSidecar
		number := rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(blockID))
		err := c.rpcClient.CallContext(ctx, &r, "eth_getBlobSidecars", number.String())
		if err == nil && r == nil {
			return nil, ethereum.NotFound
		}
		sidecars := make([]*structs.Sidecar, 0)
		idx := 0
		for _, b := range r {
			for j := range b.BlobSidecar.Blobs {
				sidecars = append(sidecars,
					&structs.Sidecar{
						Index:         strconv.Itoa(idx),
						Blob:          b.BlobSidecar.Blobs[j],
						KzgCommitment: b.BlobSidecar.Commitments[j],
						KzgProof:      b.BlobSidecar.Proofs[j],
					},
				)
				idx++
			}
		}
		return sidecars, err
	}
	return c.beaconClient.GetBlob(ctx, blockID)
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

// Define the Go structs to match the JSON structure
type BlobSidecar struct {
	Blobs       []string `json:"blobs"`
	Commitments []string `json:"commitments"`
	Proofs      []string `json:"proofs"`
}

type BlobTxSidecar struct {
	BlobSidecar BlobSidecar `json:"blobSidecar"`
	BlockNumber string      `json:"blockNumber"`
	BlockHash   string      `json:"blockHash"`
	TxIndex     string      `json:"txIndex"`
	TxHash      string      `json:"txHash"`
}
