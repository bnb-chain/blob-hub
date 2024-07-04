package cmn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	getObjectPath = "/greenfield/storage/head_object/%s/%s" // bucketName, objectName
	getParamsPath = "/greenfield/storage/params"
)

type ChainClient struct {
	hc   *http.Client
	host string
}

func NewChainClient(host string) (*ChainClient, error) {
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
	return &ChainClient{hc: client, host: host}, nil
}

func (c *ChainClient) GetObjectMeta(ctx context.Context, bucketName, objectName string) (*ObjectInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.host+fmt.Sprintf(getObjectPath, bucketName, objectName), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK response status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	getObjectResp := GetObjectInfoResponse{}
	err = json.Unmarshal(body, &getObjectResp)
	if err != nil {
		return nil, err
	}
	return &getObjectResp.ObjectInfo, nil
}

func (c *ChainClient) GetParams(ctx context.Context) (*VersionedParams, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.host+getParamsPath, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK response status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	getParamsResp := GetParamsResponse{}
	err = json.Unmarshal(body, &getParamsResp)
	if err != nil {
		return nil, err
	}

	return &getParamsResp.Params.VersionedParams, nil
}
