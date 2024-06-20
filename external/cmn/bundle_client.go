package cmn

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	modle "github.com/node-real/greenfield-bundle-service/models"
	"github.com/node-real/greenfield-bundle-service/types"

	bundlesdk "github.com/bnb-chain/greenfield-bundle-sdk/bundle"
	bundlesdktypes "github.com/bnb-chain/greenfield-bundle-sdk/types"
)

const (
	pathCreateBundle   = "/v1/createBundle"
	pathFinalizeBundle = "/v1/finalizeBundle"
	pathDeleteBundle   = "/v1/deleteBundle"
	pathUploadBundle   = "/v1/uploadBundle"

	pathUploadObject    = "/v1/uploadObject"
	pathGetBundleInfo   = "/v1/queryBundle/%s/%s"
	pathGetBundleObject = "/v1/view/%s/%s/%s" // {bucketName}/{bundleName}/{objectName}

	bundleExpiredTime = 24 * time.Hour
)

var (
	ErrorBundleNotExist       = errors.New("the bundle not exist in bundle service")
	ErrorBundleObjectNotExist = errors.New("the bundle object not exist in bundle service")
)

type BundleClientOption interface {
	Apply(*BundleClient)
}

type BundleClientOptionFunc func(*BundleClient)

// Apply set up the option field to the client instance.
func (f BundleClientOptionFunc) Apply(client *BundleClient) {
	f(client)
}

func WithPrivateKey(privateKey []byte) BundleClientOption {
	return BundleClientOptionFunc(func(client *BundleClient) {
		client.privKey = privateKey
	})
}

type BundleClient struct {
	hc      *http.Client
	host    string
	privKey []byte
	addr    common.Address
}

func NewBundleClient(host string, opts ...BundleClientOption) (*BundleClient, error) {
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
	bundleClient := &BundleClient{hc: client,
		host: host,
	}
	for _, opt := range opts {
		opt.Apply(bundleClient)
	}
	if len(bundleClient.privKey) != 0 {
		privateKey, err := crypto.ToECDSA(bundleClient.privKey)
		if err != nil {
			return nil, err
		}
		bundleClient.addr = crypto.PubkeyToAddress(privateKey.PublicKey)
	}
	return bundleClient, nil
}

func (c *BundleClient) CreateBundle(bundleName, bucketName string) error {
	headers := map[string]string{
		"Content-Type":              "application/json",
		"X-Bundle-Bucket-Name":      bucketName,
		"X-Bundle-Name":             bundleName,
		"X-Bundle-Expiry-Timestamp": fmt.Sprintf("%d", time.Now().Add(1*time.Hour).Unix()),
	}
	resp, err := c.sendRequest(c.host+pathCreateBundle, "POST", headers, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bodyStr, err := ReadResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK response status: %s, err %s", resp.Status, bodyStr)
	}
	return nil
}

func (c *BundleClient) FinalizeBundle(bundleName, bucketName string) error {
	headers := map[string]string{
		"Content-Type":              "application/json",
		"X-Bundle-Bucket-Name":      bucketName,
		"X-Bundle-Name":             bundleName,
		"X-Bundle-Expiry-Timestamp": fmt.Sprintf("%d", time.Now().Add(bundleExpiredTime).Unix()),
	}
	// finalize bundle
	resp, err := c.sendRequest(c.host+pathFinalizeBundle, "POST", headers, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bodyStr, err := ReadResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK response status: %s, err %s", resp.Status, bodyStr)
	}
	return nil
}

func (c *BundleClient) DeleteBundle(bundleName, bucketName string) error {
	headers := map[string]string{
		"Content-Type":              "application/json",
		"X-Bundle-Bucket-Name":      bucketName,
		"X-Bundle-Name":             bundleName,
		"X-Bundle-Expiry-Timestamp": fmt.Sprintf("%d", time.Now().Add(bundleExpiredTime).Unix()),
	}
	resp, err := c.sendRequest(c.host+pathDeleteBundle, "POST", headers, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bodyStr, err := ReadResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK response status: %s, err %s", resp.Status, bodyStr)
	}
	return nil
}

func (c *BundleClient) UploadAndFinalizeBundle(bundleName, bucketName, bundleDir, bundlePath string) error {

	bundleObject, _, err := bundleDirectory(bundleDir)
	if err != nil {
		return err
	}
	bundleFilePath := bundlePath
	err = saveBundleToFile(bundleObject, bundleFilePath)
	if err != nil {
		return err
	}

	bundleFile, err := os.Open(bundleFilePath)
	if err != nil {
		return err
	}
	defer bundleFile.Close()
	bundleContent, err := io.ReadAll(bundleFile)
	if err != nil {
		return err
	}

	// Calculate the SHA256 hash of the bundle file content
	hash := sha256.Sum256(bundleContent)
	hashInHex := hex.EncodeToString(hash[:])

	// Create a new multipart form and add the bundle file to it
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	filePart, err := writer.CreateFormFile("file", bundleFile.Name())
	if err != nil {
		return err
	}
	_, err = io.Copy(filePart, bytes.NewReader(bundleContent))
	if err != nil {
		return err
	}
	err = writer.Close()
	if err != nil {
		return err
	}
	headers := map[string]string{
		"Content-Type":              writer.FormDataContentType(),
		"X-Bundle-Bucket-Name":      bucketName,
		"X-Bundle-Name":             bundleName,
		"X-Bundle-File-Sha256":      hashInHex,
		"X-Bundle-Expiry-Timestamp": fmt.Sprintf("%d", time.Now().Add(bundleExpiredTime).Unix()),
	}
	resp, err := c.sendRequest(c.host+pathUploadBundle, "POST", headers, body.Bytes())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bodyStr, err := ReadResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK response status: %s, err %s", resp.Status, bodyStr)
	}
	return nil
}

func (c *BundleClient) UploadObject(fileName, bucketName, bundleName, contentType string, file *os.File) error {
	// CreateBlock a new SHA256 hash
	hash := sha256.New()

	// Write the file's content to the hash
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	// Get the hash sum in bytes
	hashInBytes := hash.Sum(nil)[:]

	// Hex encode the hash sum
	hashInHex := hex.EncodeToString(hashInBytes)

	// Reset the file read pointer to the beginning
	_, _ = file.Seek(0, 0)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	filePart, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return err
	}
	_, err = io.Copy(filePart, file)
	if err != nil {
		return err
	}
	err = writer.Close()
	if err != nil {
		return err
	}
	headers := map[string]string{
		"Content-Type":              writer.FormDataContentType(),
		"X-Bundle-Bucket-Name":      bucketName,
		"X-Bundle-Name":             bundleName,
		"X-Bundle-File-Name":        fileName,
		"X-Bundle-Content-Type":     contentType,
		"X-Bundle-Expiry-Timestamp": fmt.Sprintf("%d", time.Now().Add(bundleExpiredTime).Unix()),
		"X-Bundle-File-Sha256":      hashInHex,
	}
	resp, err := c.sendRequest(c.host+pathUploadObject, "POST", headers, body.Bytes())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bodyStr, err := ReadResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK response status: %s, err %s", resp.Status, bodyStr)
	}
	return nil
}

func (c *BundleClient) GetBundleInfo(bucketName, bundleName string) (*modle.QueryBundleResponse, error) {
	req, err := http.NewRequest("GET", c.host+fmt.Sprintf(pathGetBundleInfo, bucketName, bundleName), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrorBundleNotExist
		}
		return nil, fmt.Errorf("received non-OK response status: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	bundle := &modle.QueryBundleResponse{}
	return bundle, json.Unmarshal(body, bundle)
}

func (c *BundleClient) GetObject(bucketName, bundleName, objectName string) (string, error) {
	path := fmt.Sprintf(pathGetBundleObject, bucketName, bundleName, objectName)
	req, err := http.NewRequest("GET", c.host+path, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return "", ErrorBundleObjectNotExist
		}
		return "", fmt.Errorf("received non-OK response status: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *BundleClient) sendRequest(url, method string, headers map[string]string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	signature, err := c.signMessage(types.TextHash(crypto.Keccak256([]byte(types.GetCanonicalRequest(req)))))
	if err != nil {
		return nil, err
	}
	req.Header.Set(types.HTTPHeaderAuthorization, hex.EncodeToString(signature))
	return c.hc.Do(req)
}

func ReadResponseBody(resp *http.Response) (string, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *BundleClient) signMessage(message []byte) ([]byte, error) {
	privateKey, err := crypto.ToECDSA(c.privKey)
	if err != nil {
		return nil, err
	}
	signature, err := crypto.Sign(message, privateKey)
	if err != nil {
		return nil, err
	}
	return signature, err
}

func bundleDirectory(dir string) (io.ReadSeekCloser, int64, error) {
	b, err := bundlesdk.NewBundle()
	if err != nil {
		return nil, 0, err
	}

	err = filepath.Walk(dir, visit(dir, b))
	if err != nil {
		return nil, 0, err
	}
	return b.FinalizeBundle()
}
func visit(root string, b *bundlesdk.Bundle) filepath.WalkFunc {
	return func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !f.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			relativePath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}

			ext := filepath.Ext(path)
			contentType := mime.TypeByExtension(ext)

			content, err := io.ReadAll(file)
			if err != nil {
				return err
			}

			hash := sha256.Sum256(content)

			options := &bundlesdktypes.AppendObjectOptions{
				ContentType: contentType,
				HashAlgo:    bundlesdktypes.HashAlgo_SHA256, // Set the hash algorithm to SHA256
				Hash:        hash[:],                        // Set the hash
			}

			_, err = file.Seek(0, io.SeekStart)
			if err != nil {
				return err
			}

			_, err = b.AppendObject(relativePath, file, options)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func saveBundleToFile(bundle io.ReadSeekCloser, filePath string) error {
	// Create a new file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Copy the contents of the bundle to the file
	_, err = io.Copy(file, bundle)
	if err != nil {
		return err
	}

	return nil
}
