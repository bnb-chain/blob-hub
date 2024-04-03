package types

import (
	"fmt"
	"strconv"
	"strings"
)

func GetBlobName(height, index uint64) string {
	return fmt.Sprintf("blob_h%d_i%d", height, index)
}

func GetBundleName(startHeight, endHeight uint64) string {
	return fmt.Sprintf("blobs_s%d_e%d", startHeight, endHeight)
}

func ParseBlobName(blobName string) (height uint64, index uint64, err error) {
	parts := strings.Split(blobName, "_")
	height, err = strconv.ParseUint(parts[1][1:], 10, 64)
	if err != nil {
		return
	}
	index, err = strconv.ParseUint(parts[2][1:], 10, 64)
	if err != nil {
		return
	}
	return
}

func ParseBundleName(bundleName string) (startHeight, endHeight uint64, err error) {
	parts := strings.Split(bundleName, "_")
	startHeight, err = strconv.ParseUint(parts[1][1:], 10, 64)
	if err != nil {
		return
	}
	endHeight, err = strconv.ParseUint(parts[2][1:], 10, 64)
	if err != nil {
		return
	}
	return
}
