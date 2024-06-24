package cmn

import (
	"context"
	"encoding/xml"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type SPClient struct {
	hc   *http.Client
	host string
}

func NewSPClient(host string) (*SPClient, error) {
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
	return &SPClient{hc: client, host: host}, nil
}

func (c *SPClient) GetBucketReadQuota(ctx context.Context, bucketName string) (QuotaInfo, error) {
	year, month, _ := time.Now().Date()
	var date string
	if int(month) < 10 {
		date = strconv.Itoa(year) + "-" + "0" + strconv.Itoa(int(month))
	} else {
		date = strconv.Itoa(year) + "-" + strconv.Itoa(int(month))
	}
	var urlStr string
	parts := strings.Split(c.host, "//")
	urlStr = parts[0] + "//" + bucketName + "." + parts[1] + "/"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return QuotaInfo{}, err
	}
	// set query parameters
	q := req.URL.Query()
	q.Add("read-quota", "")
	q.Add("year-month", date)
	req.URL.RawQuery = q.Encode()
	resp, err := c.hc.Do(req)
	if err != nil {
		return QuotaInfo{}, err
	}
	defer resp.Body.Close()
	QuotaResult := QuotaInfo{}
	err = xml.NewDecoder(resp.Body).Decode(&QuotaResult)
	if err != nil {
		return QuotaInfo{}, err
	}
	return QuotaResult, nil
}
