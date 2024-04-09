package external

import (
	"encoding/hex"
	"fmt"
	"os"
	"testing"
)

const NodeRealBundle = "https://gnfd-testnet-bundle.nodereal.io"
const PrivateKeyHex = "d65f7cf21fe3eff9feef1dd86cea6bae8a30f6c26830e734d628c78e80debfd5"

func TestGetBundleInfo(t *testing.T) {
	bundleClient, err := NewBundleClient(NodeRealBundle)
	if err != nil {
		panic(err)
	}
	bundleInfo, err := bundleClient.GetBundleInfo("bsc-blobs", "mybundle3")
	if err != nil {
		panic(err)
	}
	fmt.Println(bundleInfo)
}

func TestGetBundleObject(t *testing.T) {

	bundleClient, err := NewBundleClient(NodeRealBundle)
	if err != nil {
		panic(err)
	}
	objectInfo, err := bundleClient.GetObject("bsc-blobs", "blobs_s8762203_e8762222", "blob-store/blob_h8762217_i4")
	if err != nil {
		panic(err)
	}
	fmt.Println(objectInfo)
}

func TestUploadObject(t *testing.T) {
	pkBz, err := hex.DecodeString(PrivateKeyHex)
	if err != nil {
		panic(err)
	}

	bundleClient, err := NewBundleClient(NodeRealBundle, WithPrivateKey(pkBz))
	if err != nil {
		panic(err)
	}
	bundleInfo, err := bundleClient.GetBundleInfo("bsc-blobs", "blobs_s8761850_e8761851")
	if err != nil {
		panic(err)
	}
	fmt.Println(bundleInfo)
	file, err := os.Open("../blob-store/blob_h8761850_i0")
	err = bundleClient.UploadObject(file.Name(), "bsc-blobs", "blobs_s8761850_e8761851", "text/plain", file)
	if err != nil {
		panic(err)
	}
}

func TestDeleteFinalizingBundle(t *testing.T) {
	pkBz, err := hex.DecodeString(PrivateKeyHex)
	if err != nil {
		panic(err)
	}
	bundleClient, err := NewBundleClient(NodeRealBundle, WithPrivateKey(pkBz))
	if err != nil {
		panic(err)
	}
	err = bundleClient.DeleteBundle("blobs_s8777090_e8777094", "bsc-blobs")
	if err != nil {
		panic(err)
	}
}

func TestCreateLocalBundle(t *testing.T) {
	pkBz, err := hex.DecodeString(PrivateKeyHex)
	if err != nil {
		panic(err)
	}
	bundleClient, err := NewBundleClient(NodeRealBundle, WithPrivateKey(pkBz))
	if err != nil {
		panic(err)
	}

	err = bundleClient.UploadAndFinalizeBundle("blobssss", "bsc-blobs", "../test/", "")
	if err != nil {
		t.Fatal(err)
	}
}
