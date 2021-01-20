package proxy

import (
	"log"
	"math/big"
	"strconv"
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

	log.Println("Nonce / Distance / TS --->", nonceDec, " / ", distance, " / ", workerTimestamp)

	if workId != t.WorkId {
		log.Printf("Stale share from %v@%v", login, ip)
		return false, false
	}

	share := Block{
   	work:        t.Header,
		number:      t.Height,
		difficulty:  big.NewInt(shareDiff),
		distance:    distance,
		nonce:       nonce,
		MinerKey:    t.MinerKey,
		MerkleRoot:  t.MerkleRoot,
		WorkId:      workId,
		WorkerTS:    workerTimestamp,
	}

	block := Block{
    work:        t.Header,
		number:      t.Height,
		difficulty:  t.Difficulty,
		distance:    distance, 
		nonce:       nonce,
		MinerKey:    t.MinerKey,
		MerkleRoot:  t.MerkleRoot,
		WorkId:      workId,
		WorkerTS:    workerTimestamp,
	}

	if !olhash.Verify(share.difficulty, share.work, share.MinerKey,
    	   	          share.MerkleRoot, share.nonce, share.WorkerTS) {
		return false, false
	}


	// prepare block submission
	params_out := []string{workId, nonceDec, t.Difficulty.String(), params[1],
      		               params[2], "0", "0"}

	if olhash.Verify(block.difficulty, block.work, block.MinerKey,
	                 block.MerkleRoot, block.nonce, block.WorkerTS) {
		ok, err := s.miningRpc().SubmitBlock(params_out)
		if err != nil {
			log.Printf("Block submission failure at height %v for %v: %v", t.Height, t.Header, err)
		} else if !ok {
			log.Printf("Block rejected at height %v for %v", t.Height, t.Header)
			return false, false
		} else {
			s.fetchBlockTemplate()
			exist, err := s.backend.WriteBlock(login, id, params_out, shareDiff, t.Difficulty.Int64(), t.Height, s.hashrateExpiration)
			if exist {
				return true, false
			}
			if err != nil {
				log.Println("Failed to insert block candidate into backend:", err)
			} else {
				log.Printf("Inserted block %v to backend", t.Height)
			}
			log.Printf("Block found by miner %v@%v at height %d", login, ip, t.Height)
		}
	} else {
		exist, err := s.backend.WriteShare(login, id, params_out, shareDiff, t.Height, s.hashrateExpiration)
		if exist {
			return true, false
		}
		if err != nil {
			log.Println("Failed to insert share data into backend:", err)
		}
	}
	return false, true
}
