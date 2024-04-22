package external

import (
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

type ETHClient struct {
	Eth1Client   *ethclient.Client
	BeaconClient *BeaconClient
	endpoint     string
	updatedAt    time.Time
}

func NewETHClient(rpcAddrs, beaconRPCAddrs string) *ETHClient {
	ethClient, err := ethclient.Dial(rpcAddrs)
	if err != nil {
		panic("new eth client error")
	}
	beaconClient, err := NewBeaconClient(beaconRPCAddrs)
	if err != nil {
		panic("new eth client error")
	}
	return &ETHClient{
		Eth1Client:   ethClient,
		BeaconClient: beaconClient,
		endpoint:     rpcAddrs,
		updatedAt:    time.Now(),
	}
}
