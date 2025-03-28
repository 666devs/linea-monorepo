package test_utils

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/consensys/compress/lzss"
	fr381 "github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	"github.com/consensys/linea-monorepo/prover/backend/execution"
	"github.com/consensys/linea-monorepo/prover/lib/compressor/blob"
	"github.com/consensys/linea-monorepo/prover/lib/compressor/blob/encode"
	v1 "github.com/consensys/linea-monorepo/prover/lib/compressor/blob/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func GenTestBlob(t require.TestingT, maxNbBlocks int) []byte {
	testBlocks, bm := TestBlocksAndBlobMaker(t)

	// Compress blocks
	cptBlock := 0
	for i, block := range testBlocks {
		// get a random from 1 to 5

		bSize := RandIntn(5) + 1 // #nosec G404 -- false positive

		if cptBlock > bSize && i%3 == 0 {
			cptBlock = 0
			bm.StartNewBatch()
		}

		appended, err := bm.Write(block, false)
		if !appended || i == maxNbBlocks {
			assert.NoError(t, err, "append a valid block should not generate an error")
			break
		} else {
			cptBlock++
		}
	}
	return bm.Bytes()
}

func LoadTestBlocks(testDataDir string) (testBlocks [][]byte, err error) {
	entries, err := os.ReadDir(testDataDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		jsonString, err := os.ReadFile(filepath.Join(testDataDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var proverInput execution.Request
		if err = json.Unmarshal(jsonString, &proverInput); err != nil {
			return nil, err
		}

		for _, block := range proverInput.Blocks() {
			var bb bytes.Buffer
			if err = block.EncodeRLP(&bb); err != nil {
				return nil, err
			}
			testBlocks = append(testBlocks, bb.Bytes())
		}
	}
	return testBlocks, nil
}

func RandIntn(n int) int { // TODO @Tabaie remove
	var b [8]byte
	_, _ = rand.Read(b[:])
	return int(binary.BigEndian.Uint64(b[:]) % uint64(n))
}

func EmptyBlob(t require.TestingT) []byte {
	var headerB bytes.Buffer

	repoRoot, err := blob.GetRepoRootPath()
	assert.NoError(t, err)
	// Init bm
	bm, err := v1.NewBlobMaker(1000, filepath.Join(repoRoot, "prover/lib/compressor/compressor_dict.bin"))
	assert.NoError(t, err)

	if _, err = bm.Header.WriteTo(&headerB); err != nil {
		panic(err)
	}

	compressor, err := lzss.NewCompressor(GetDict(t))
	assert.NoError(t, err)

	var bb bytes.Buffer
	if _, err = encode.PackAlign(&bb, headerB.Bytes(), fr381.Bits-1, encode.WithAdditionalInput(compressor.Bytes())); err != nil {
		panic(err)
	}
	return bb.Bytes()
}

func SingleBlockBlob(t require.TestingT) []byte {
	testBlocks, bm := TestBlocksAndBlobMaker(t)

	ok, err := bm.Write(testBlocks[0], false)
	assert.NoError(t, err)
	assert.True(t, ok)

	return bm.Bytes()
}

// TinyTwoBatchBlob produces a blob with two batches, each consisting of one block
func TinyTwoBatchBlob(t require.TestingT) []byte {

	testBlocks, bm := TestBlocksAndBlobMaker(t)

	for _, block := range testBlocks[:2] {
		ok, err := bm.Write(block, false)
		assert.NoError(t, err)
		assert.True(t, ok)
		bm.StartNewBatch()
	}

	return bm.Bytes()
}

// ConsecutiveBlobs (t, i, j, k) produces three blobs from the ranges [0:i], [i:i+j], [i+j:i+j+k] of test blocks
// each block is in its own batch
func ConsecutiveBlobs(t require.TestingT, n ...int) [][]byte {
	testBlocks, bm := TestBlocksAndBlobMaker(t)

	res := make([][]byte, 0, len(n))
	for _, n := range n {
		for j := 0; j < n; j++ {
			ok, err := bm.Write(testBlocks[j], false)
			assert.NoError(t, err)
			assert.True(t, ok)
			bm.StartNewBatch()
		}
		testBlocks = testBlocks[n:]
		res = append(res, bm.Bytes())
		bm.Reset()
	}

	return res
}

func TestBlocksAndBlobMaker(t require.TestingT) ([][]byte, *v1.BlobMaker) {
	repoRoot, err := blob.GetRepoRootPath()
	assert.NoError(t, err)
	testBlocks, err := LoadTestBlocks(filepath.Join(repoRoot, "testdata/prover-v2/prover-execution/requests"))
	assert.NoError(t, err)
	// Init bm
	bm, err := v1.NewBlobMaker(40000, filepath.Join(repoRoot, "prover/lib/compressor/compressor_dict.bin"))
	assert.NoError(t, err)
	return testBlocks, bm
}

func GetDict(t require.TestingT) []byte {
	dict, err := blob.GetDict()
	require.NoError(t, err)
	return dict
}
