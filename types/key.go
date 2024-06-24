package types

import (
	"fmt"
	"strconv"
	"strings"
)

const RootLength = 32

func GetBlobName(slot uint64, index int) string {
	return fmt.Sprintf("blob_h%d_i%d", slot, index)
}

func GetBundleName(startSlot, endSlot uint64) string {
	return fmt.Sprintf("blobs_s%d_e%d", startSlot, endSlot)
}

func ParseBlobName(blobName string) (slot uint64, index uint64, err error) {
	parts := strings.Split(blobName, "_")
	slot, err = strconv.ParseUint(parts[1][1:], 10, 64)
	if err != nil {
		return
	}
	index, err = strconv.ParseUint(parts[2][1:], 10, 64)
	if err != nil {
		return
	}
	return
}

func ParseBundleName(bundleName string) (startSlot, endSlot uint64, err error) {
	parts := strings.Split(bundleName, "_")
	startSlot, err = strconv.ParseUint(parts[1][1:], 10, 64)
	if err != nil {
		return
	}
	endSlot, err = strconv.ParseUint(parts[2][1:], 10, 64)
	if err != nil {
		return
	}
	return
}
