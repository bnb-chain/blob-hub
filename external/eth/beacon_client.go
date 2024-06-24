package eth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
)

var (
	ErrBlockNotFound = errors.New("the block is not found in beacon chain") // note: get a forked block also return 404
)

const (
	pathGetSidecars = "/eth/v1/beacon/blob_sidecars/%s"
	pathGetBlock    = "/eth/v2/beacon/blocks/%s"
	pathGetHeader   = "/eth/v1/beacon/headers/%s"
)

type BeaconClient struct {
	hc   *http.Client
	host string
}

// NewBeaconClient returns a new beacon client.
func NewBeaconClient(host string) (*BeaconClient, error) {
	transport := &http.Transport{
		DisableCompression:  true,
		MaxIdleConnsPerHost: 1000,
		MaxConnsPerHost:     1000,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{
		Timeout:   10 * time.Minute,
		Transport: transport,
	}
	return &BeaconClient{hc: client, host: host}, nil
}

func (c *BeaconClient) GetBlob(ctx context.Context, slotNumber uint64) ([]*structs.Sidecar, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.host+fmt.Sprintf(pathGetSidecars, strconv.FormatUint(slotNumber, 10)), nil)
	if err != nil {
		return nil, err
	}
	r, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = r.Body.Close()
	}()
	respBz, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading http response body %s", err)
	}
	var sidecars structs.SidecarsResponse
	err = json.Unmarshal(respBz, &sidecars)
	if err != nil {
		return nil, err
	}
	return sidecars.Data, nil
}

func (c *BeaconClient) GetBeaconBlock(ctx context.Context, slotNumber uint64) (*structs.GetBlockV2Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.host+fmt.Sprintf(pathGetBlock, strconv.FormatUint(slotNumber, 10)), nil)
	if err != nil {
		return nil, err
	}
	r, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = r.Body.Close()
	}()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading http response body %s", err)
	}

	if r.StatusCode != http.StatusOK {
		if r.StatusCode == http.StatusNotFound {
			return nil, ErrBlockNotFound
		}
		return nil, fmt.Errorf("received non-OK response status: %s", r.Status)
	}
	resp := &structs.GetBlockV2Response{}
	return resp, json.Unmarshal(b, resp)

}

func (c *BeaconClient) GetLatestBeaconBlock(ctx context.Context) (*structs.GetBlockV2Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.host+fmt.Sprintf(pathGetBlock, "head"), nil)
	if err != nil {
		return nil, err
	}
	r, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = r.Body.Close()
	}()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading http response body %s", err)
	}
	resp := &structs.GetBlockV2Response{}
	return resp, json.Unmarshal(b, resp)

}

func (c *BeaconClient) GetBeaconHeader(ctx context.Context, slotNumber uint64) (*structs.GetBlockHeaderResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.host+fmt.Sprintf(pathGetHeader, strconv.FormatUint(slotNumber, 10)), nil)
	if err != nil {
		return nil, err
	}
	r, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = r.Body.Close()
	}()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading http response body %s", err)
	}

	if r.StatusCode != http.StatusOK {
		if r.StatusCode == http.StatusNotFound {
			return nil, ErrBlockNotFound
		}
		return nil, fmt.Errorf("received non-OK response status: %s", r.Status)
	}
	resp := &structs.GetBlockHeaderResponse{}
	return resp, json.Unmarshal(b, resp)

}
