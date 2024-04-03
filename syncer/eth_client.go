package syncer

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"time"
)

type ETHClient struct {
	eth1Client   *ethclient.Client
	beaconClient *BeaconClient
	endpoint     string
	updatedAt    time.Time
}

func newETHClient(rpcAddrs, beaconAddrs string) *ETHClient {
	ethClient, err := ethclient.Dial(rpcAddrs)
	if err != nil {
		panic("new eth client error")
	}
	beaconClient, err := NewBeaconClient(beaconAddrs, time.Second*3)
	if err != nil {
		panic("new eth client error")
	}
	return &ETHClient{
		eth1Client:   ethClient,
		beaconClient: beaconClient,
		endpoint:     rpcAddrs,
		updatedAt:    time.Now(),
	}
}
