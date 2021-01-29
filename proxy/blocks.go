package proxy

import (
	"log"
	"math/big"
	"encoding/json"
	"strconv"
	//"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"github.com/lgray/open-overline-pool/rpc"
	//"github.com/lgray/open-overline-pool/util"
)

const maxBacklog = 3

type heightDiffPair struct {
	diff   *big.Int
	height uint64
  Work string
  MinerKey string
  MerkleRoot string
  Timestamp int64
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
	MinerKey             string
	MerkleRoot           string
	WorkId               string
  Timestamp            int64
  LastBlockHash        string
  BlockIsSubmitted     bool
}

type Block struct {
	difficulty  *big.Int
	work        string
	nonce       uint64
	distance    uint64
	number      uint64
	MinerKey    string
	MerkleRoot  string
	WorkId      string
	WorkerTS    int64
	hashNoNonce common.Hash
	mixDigest   common.Hash
}

func (b Block) Difficulty() *big.Int     { return b.difficulty }
func (b Block) HashNoNonce() common.Hash { return b.hashNoNonce }
func (b Block) Nonce() uint64            { return b.nonce }
func (b Block) MixDigest() common.Hash   { return b.mixDigest }
func (b Block) NumberU64() uint64        { return b.number }
func (b Block) Work() string             { return b.work }

func (s *ProxyServer) fetchBlockTemplate() {
	r := s.rpc()
	r_mine := s.miningRpc()
	t := s.currentBlockTemplate()
	reply, err := r_mine.GetWork()

  blockissubmitted, _ := strconv.ParseBool(reply[8])
  
  if len(reply) == 0 || len(reply[0]) == 0 {
    log.Printf("No block template from node yet!")
    return
  }
  
	if err != nil {
		log.Printf("Error while refreshing block template on %s: %s", r.Name, err)
		return
	}
	// No need to update, we have fresh job
	if t != nil && t.Header == reply[0] {
    // if block completed state updated change our template
    if blockissubmitted && blockissubmitted != t.BlockIsSubmitted {
      update := *t
      update.BlockIsSubmitted = blockissubmitted
      s.blockTemplate.Store(&update)
      log.Printf("block %v:%v template marked completed!", t.WorkId, t.Header)
    } 
		return
	}
	diff := new(big.Int)
	diff.SetString(reply[2], 10)
	//log.Println(diff)
	height, err := strconv.ParseUint(string(reply[3]), 10, 64)

  thetimestamp, _ := strconv.ParseInt(reply[6], 10, 64)
    
	pendingReply := &rpc.GetBlockReplyPart{
		Difficulty: strconv.FormatInt(s.config.Proxy.Difficulty, 10),
		Number:     json.Number(reply[3]),
	}

	newTemplate := BlockTemplate{
		Header:               reply[0],
		Seed:                 reply[1],
		Target:               reply[2],
		Height:               height,
		Difficulty:           diff,
		GetPendingBlockCache: pendingReply,
		headers:              make(map[string]heightDiffPair),
		MinerKey:             reply[5],
		MerkleRoot:           reply[1],
		WorkId:               reply[4],
    Timestamp:            thetimestamp,
    LastBlockHash:        reply[7],
    BlockIsSubmitted:     blockissubmitted,
	}
	// Copy job backlog and add current one
	newTemplate.headers[reply[4]] = heightDiffPair{
		diff:       diff,
		height:     height,
    Work:       reply[0],
    MinerKey:   reply[5],
    MerkleRoot: reply[1],
    Timestamp:  thetimestamp,
	}
	if t != nil {
		for k, v := range t.headers {
			if v.height > height-maxBacklog {
				newTemplate.headers[k] = v
			}
		}
	}
	s.blockTemplate.Store(&newTemplate)
	log.Printf("New block to mine on %s at height %d / %s / %d / %d", r.Name, height, reply[0][0:10], diff, thetimestamp)

	// Stratum
	if s.config.Proxy.Stratum.Enabled {
		go s.broadcastNewJobs()
	}
}
