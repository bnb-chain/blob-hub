package external

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"time"
)

type ETHClient struct {
	Eth1Client   *ethclient.Client
	BeaconClient *BeaconClient
	endpoint     string
	updatedAt    time.Time
}

func NewETHClient(rpcAddrs, beaconAddrs string) *ETHClient {
	ethClient, err := ethclient.Dial(rpcAddrs)
	if err != nil {
		panic("new eth client error")
	}
	beaconClient, err := NewBeaconClient(beaconAddrs, time.Second*3)
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
