package proxy

import (
	"log"
	"math/big"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"github.com/zano-mining/open-zano-pool/rpc"
)

const maxBacklog = 3

type heightDiffPair struct {
	diff   *big.Int
	height uint64
}

type BlockTemplate struct {
	sync.RWMutex
	Header               string
	Seed                 string
	Target               string
	Difficulty           *big.Int
	Height               uint64
	GetPendingBlockCache *rpc.GetBlockReplyPart
	nonces               map[string]bool
	headers              map[string]heightDiffPair
}

type Block struct {
	difficulty  *big.Int
	hashNoNonce common.Hash
	nonce       uint64
	mixDigest   common.Hash
	number      uint64
}

func (b Block) Difficulty() *big.Int     { return b.difficulty }
func (b Block) HashNoNonce() common.Hash { return b.hashNoNonce }
func (b Block) Nonce() uint64            { return b.nonce }
func (b Block) MixDigest() common.Hash   { return b.mixDigest }
func (b Block) NumberU64() uint64        { return b.number }

func (s *ProxyServer) fetchBlockTemplate() {
	r := s.rpc()
	t := s.currentBlockTemplate()
	reply, err := r.GetWork(s.config.Proxy.Address)
	if err != nil {
		log.Printf("Error while refreshing block template on %s: %s", r.Name, err)
		return
	}
	// No need to update, we have fresh job
	if t != nil && t.Header == reply.Blob {
		return
	}
	diff, _ := new(big.Int).SetString(reply.Difficulty, 10)
	height := reply.Height

	pendingReply := &rpc.GetBlockReplyPart{
		Difficulty: strconv.FormatInt(s.config.Proxy.Difficulty, 10),
		Number:     strconv.FormatUint(reply.Height, 10),
	}

	newTemplate := BlockTemplate{
		Header:               reply.Blob,
		Seed:                 reply.Seed,
		Target:               reply.PrevHash,
		Height:               height,
		Difficulty:           diff,
		GetPendingBlockCache: pendingReply,
		headers:              make(map[string]heightDiffPair),
	}
	// Copy job backlog and add current one
	newTemplate.headers[reply.Blob] = heightDiffPair{
		diff:   diff,
		height: height,
	}
	if t != nil {
		for k, v := range t.headers {
			if v.height > height-maxBacklog {
				newTemplate.headers[k] = v
			}
		}
	}
	s.blockTemplate.Store(&newTemplate)
	log.Printf("New block to mine on %s at height %d / %s / %d", r.Name, height, reply.Blob[0:10], diff)

	// Stratum
	if s.config.Proxy.Stratum.Enabled {
		go s.broadcastNewJobs()
	}
}
