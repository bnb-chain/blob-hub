package util

import (
	"bytes"
	"crypto/sha256"
	"io"
	"sync"

	"github.com/klauspost/reedsolomon"
)

// The following functions are used to compute the integrity hash of the data, imported from the greenfield-common package.

// ComputeIntegrityHashSerial computes the integrity hash of the data serially
func ComputeIntegrityHashSerial(reader io.Reader, segmentSize int64, dataShards, parityShards int) ([][]byte, int64, error) {
	var segChecksumList [][]byte
	ecShards := dataShards + parityShards

	encodeDataHash := make([][][]byte, ecShards)
	for i := 0; i < ecShards; i++ {
		encodeDataHash[i] = make([][]byte, 0)
	}

	hashList := make([][]byte, ecShards+1)
	contentLen := int64(0)
	// read the data by segment segmentSize
	for {
		seg := make([]byte, segmentSize)
		n, err := reader.Read(seg)
		if err != nil {
			if err != io.EOF {
				return nil, 0, err
			}
			break
		}

		if n > 0 && n <= int(segmentSize) {
			contentLen += int64(n)
			data := seg[:n]
			// compute segment hash
			checksum := GenerateChecksum(data)
			segChecksumList = append(segChecksumList, checksum)

			if err = encodeAndComputeHash(encodeDataHash, data, dataShards, parityShards); err != nil {
				return nil, 0, err
			}
		}
	}

	// combine the hash root of pieces of the PrimarySP
	hashList[0] = GenerateIntegrityHash(segChecksumList)

	// compute the integrity hash of the SecondarySP
	wg := &sync.WaitGroup{}
	spLen := len(encodeDataHash)
	wg.Add(spLen)
	for spID, content := range encodeDataHash {
		go func(data [][]byte, id int) {
			defer wg.Done()
			hashList[id+1] = GenerateIntegrityHash(data)
		}(content, spID)
	}

	wg.Wait()

	return hashList, contentLen, nil
}

// GenerateIntegrityHash generates integrity hash of all piece data checksum
func GenerateIntegrityHash(checksumList [][]byte) []byte {
	hash := sha256.New()
	checksumBytesTotal := bytes.Join(checksumList, []byte(""))
	hash.Write(checksumBytesTotal)
	return hash.Sum(nil)
}

// GenerateChecksum generates the checksum of one piece data
func GenerateChecksum(pieceData []byte) []byte {
	hash := sha256.New()
	hash.Write(pieceData)
	return hash.Sum(nil)
}

// encodeAndComputeHash encode the segment and compute the hash of pieces
func encodeAndComputeHash(encodeDataHash [][][]byte, segment []byte, dataShards, parityShards int) error {
	// get erasure encode bytes
	encodeShards, err := EncodeRawSegment(segment, dataShards, parityShards)
	if err != nil {
		return err
	}

	for index, shard := range encodeShards {
		// compute hash of pieces
		piecesHash := GenerateChecksum(shard)
		encodeDataHash[index] = append(encodeDataHash[index], piecesHash)
	}

	return nil
}

// EncodeRawSegment encode a raw byte array and return erasure encoded shards in orders
func EncodeRawSegment(content []byte, dataShards, parityShards int) ([][]byte, error) {
	encoder, err := NewRSEncoder(dataShards, parityShards, int64(len(content)))
	if err != nil {
		return nil, err
	}
	shards, err := encoder.EncodeData(content)
	if err != nil {
		return nil, err
	}
	return shards, nil
}

// EncodeData encodes the given data and returns the reed-solomon encoded shards
func (r *RSEncoder) EncodeData(content []byte) ([][]byte, error) {
	if len(content) == 0 {
		return make([][]byte, r.dataShards+r.parityShards), nil
	}
	encoded, err := r.encoder().Split(content)
	if err != nil {
		return nil, err
	}
	if err = r.encoder().Encode(encoded); err != nil {
		return nil, err
	}
	return encoded, nil
}

func NewRSEncoder(dataShards, parityShards int, blockSize int64) (r RSEncoder, err error) {
	// Check the parameters for sanity now.
	if dataShards <= 0 || parityShards < 0 {
		return r, reedsolomon.ErrInvShardNum
	}

	if dataShards+parityShards > 256 {
		return r, reedsolomon.ErrMaxShardNum
	}

	r = RSEncoder{
		dataShards:   dataShards,
		parityShards: parityShards,
		blockSize:    blockSize,
	}
	var encoder reedsolomon.Encoder
	var once sync.Once
	r.encoder = func() reedsolomon.Encoder {
		once.Do(func() {
			r, _ := reedsolomon.New(dataShards, parityShards,
				reedsolomon.WithAutoGoroutines(int(r.ShardSize())))
			encoder = r
		})
		return encoder
	}
	return
}

// RSEncoder - reedSolomon RSEncoder encoding details.
type RSEncoder struct {
	encoder                  func() reedsolomon.Encoder
	dataShards, parityShards int
	blockSize                int64 // the data size to be encoded
}

// ShardSize returns the size of each shard
func (r *RSEncoder) ShardSize() int64 {
	shardNum := int64(r.dataShards)
	n := r.blockSize / shardNum
	if r.blockSize > 0 && r.blockSize%shardNum != 0 {
		n++
	}
	return n
}
