package proxy

import (
	"log"
	"math/big"
	"strconv"
  "encoding/hex"
	//"strings"

	//"github.com/ethereum/ethash"
	//"github.com/ethereum/go-ethereum/common"
	"github.com/lgray/open-overline-pool/olhash"
)

func (s *ProxyServer) processShare(login, id, ip string, t *BlockTemplate, params []string) (bool, bool) {
     	log.Println("got params:", params)
     	nonceDec := params[0]
	distance, _ := strconv.ParseUint(params[1], 10, 64)
	workerTimestamp, _ := strconv.ParseInt(params[2], 10, 64)
	workId := params[3]
	nonce, _ := strconv.ParseUint(nonceDec, 10, 64)
	shareDiff := s.config.Proxy.Difficulty

  h, ok := t.headers[workId] // allow us to collect workids for the last three blocks
  
	if !ok {
		log.Printf("Stale share from %v@%v - %v", login, ip, workId)
		return false, false
	}

  if workerTimestamp != h.Timestamp {
    log.Println("Worker timestamp different from header timestamp ", workerTimestamp, " != ", h.Timestamp)
  }
  
	share := Block{
   	work:        h.Work,
		number:      h.height,
		difficulty:  big.NewInt(shareDiff),
		distance:    distance,
		nonce:       nonce,
		MinerKey:    h.MinerKey,
		MerkleRoot:  h.MerkleRoot,
		WorkId:      workId,
		WorkerTS:    h.Timestamp,
	}

	block := Block{
    work:        h.Work,
		number:      h.height,
		difficulty:  h.diff,
		distance:    distance, 
		nonce:       nonce,
		MinerKey:    h.MinerKey,
		MerkleRoot:  h.MerkleRoot,
		WorkId:      workId,
		WorkerTS:    h.Timestamp,
	}

	if !olhash.Verify(share.difficulty, share.work, share.MinerKey,
    	   	          share.MerkleRoot, share.nonce, share.WorkerTS) {
		return false, false
	}


	// prepare block submission
  bhash := []string{hex.EncodeToString(olhash.Blake2blFromBytes([]byte(t.LastBlockHash + t.MerkleRoot)))}
  params_out := []string{workId, nonceDec, h.diff.String(), params[1],
                         params[2], "0","0"}
  params_db := append(params_out, bhash...)

  // we do *not* want to submit old work IDs to BC, really bad idea
  // but we should record them as shares
  should_submit_as_block := ((h.height == t.Height) && (!t.BlockIsSubmitted))

  valid_block := olhash.Verify(new(big.Int).Add(new(big.Int).SetInt64(2500000000000), block.difficulty), block.work, block.MinerKey,
                               block.MerkleRoot, block.nonce, block.WorkerTS)
  
  if !should_submit_as_block && valid_block {
    log.Printf("Valid block solution %v:%v came after already previously solution, saving shares only!", block.work, block.nonce)
  }
  
	if (should_submit_as_block && valid_block) {

		ok, err := s.miningRpc().SubmitBlock(params_out)
		if err != nil {
			log.Printf("Block submission failure at height %v for %v: %v", h.height, t.Header, err)
		} else if !ok {
			log.Printf("Block rejected at height %v for %v", h.height, t.Header)
			return false, false
		} else {
			s.fetchBlockTemplate()
			exist, err := s.backend.WriteBlock(login, id, params_db, shareDiff,
                                         h.diff.Int64(), h.height, s.hashrateExpiration)
			if exist {
				return true, false
			}
			if err != nil {
				log.Println("Failed to insert block candidate into backend:", err)
			} else {
				log.Printf("Inserted block %v to backend",h.height)
			}
			log.Printf("Block found by miner %v@%v at height %d", login, ip, h.height)
		}
	} else {
		exist, err := s.backend.WriteShare(login, id, params_db, shareDiff, h.height, s.hashrateExpiration)
		if exist {
			return true, false
		}
		if err != nil {
			log.Println("Failed to insert share data into backend:", err)
		}
	}
	return false, true
}
