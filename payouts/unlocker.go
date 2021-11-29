package payouts

import (
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"
  "os"

	"github.com/ethereum/go-ethereum/common/math"

	"github.com/lgray/open-overline-pool/rpc"
	"github.com/lgray/open-overline-pool/storage"
	"github.com/lgray/open-overline-pool/util"
)

type UnlockerConfig struct {
	Enabled        bool    `json:"enabled"`
	PoolFee        float64 `json:"poolFee"`
	PoolFeeAddress string  `json:"poolFeeAddress"`
	Donate         bool    `json:"donate"`
	Depth          int64   `json:"depth"`
	ImmatureDepth  int64   `json:"immatureDepth"`
	KeepTxFees     bool    `json:"keepTxFees"`
	Interval       string  `json:"interval"`
	Daemon         string  `json:"daemon"`
  SCookie        string  `json:"scookie"`
	Timeout        string  `json:"timeout"`
}

const minDepth = 16

var afterTargetReward = math.MustParseBig256("1000000000000000000")

// Donate 10% from pool fees to developers
const donationFee = 10.0
const donationAccount = "0xf34fa87db39d15471bebe997860dcd49fc259318"

type BlockUnlocker struct {
	config   *UnlockerConfig
	backend  *storage.RedisClient
	rpc      *rpc.RPCClient
	halt     bool
	lastFail error
}

func NewBlockUnlocker(cfg *UnlockerConfig, backend *storage.RedisClient) *BlockUnlocker {
  poolFeeAddress := os.Getenv(cfg.PoolFeeAddress)
	if len(poolFeeAddress) != 0 && !util.IsValidHexAddress(poolFeeAddress) {
		log.Fatalln("Invalid poolFeeAddress", poolFeeAddress)
	}
	if cfg.Depth < minDepth*2 {
		log.Fatalf("Block maturity depth can't be < %v, your depth is %v", minDepth*2, cfg.Depth)
	}
	if cfg.ImmatureDepth < minDepth {
		log.Fatalf("Immature depth can't be < %v, your depth is %v", minDepth, cfg.ImmatureDepth)
	}
	u := &BlockUnlocker{config: cfg, backend: backend}
  SCookie := os.Getenv(cfg.SCookie)
	u.rpc = rpc.NewRPCClient("BlockUnlocker", cfg.Daemon, SCookie, cfg.Timeout)
	return u
}

func (u *BlockUnlocker) Start() {
	log.Println("Starting block unlocker")
	intv := util.MustParseDuration(u.config.Interval)
	timer := time.NewTimer(intv)
	log.Printf("Set block unlock interval to %v", intv)

	// Immediately unlock after start
	u.unlockPendingBlocks()
	u.unlockAndCreditMiners()
	timer.Reset(intv)

	go func() {
		for {
			select {
			case <-timer.C:
				u.unlockPendingBlocks()
				u.unlockAndCreditMiners()
				timer.Reset(intv)
			}
		}
	}()
}

type UnlockResult struct {
	maturedBlocks  []*storage.BlockData
	orphanedBlocks []*storage.BlockData
	orphans        int
	uncles         int
	blocks         int
}

func (u *BlockUnlocker) unlockCandidates(candidates []*storage.BlockData) (*UnlockResult, error) {
	result := &UnlockResult{}

	// Data row is: "height:nonce:powHash:mixDigest:timestamp:diff:totalShares"
	for _, candidate := range candidates {
		orphan := true

  	height := candidate.Height
    hash := candidate.Hash
    
  	if height < 0 {
			continue
		}

    nextHeight := height + 1
  
    blockByHash, errHash := u.rpc.GetBlockByHash(hash)
		nextBlockByHeight, errHeight := u.rpc.GetBlockByHeight(nextHeight)

    //log.Println("---- block by height ----")
    //log.Println(blockByHeight)
    //log.Println("---- block by hash ----")
    //log.Println(blockByHash)
  
		if errHeight != nil {
			log.Printf("Error while retrieving block %v from node: %v", height, errHeight)
			return nil, errHeight
		}
    if errHash != nil {
      log.Printf("Error while retrieving block %v from node: %v ", hash, errHash)
      //return nil, errHash
    }
		if nextBlockByHeight == nil {
			return nil, fmt.Errorf("Error while retrieving block %v from node, wrong node height", height)
		}
    if blockByHash == nil {
      log.Printf("Error while retrieving block %v from node, wrong node hash", hash)
    }

    hashFound := matchCandidate(blockByHash, candidate)
    foundAsPreviousHash := matchCandidateByPreviousHash(nextBlockByHeight, candidate)
  
		if foundAsPreviousHash && hashFound { // it is a reward block
			orphan = false
			result.blocks++

  		err := u.handleBlock(blockByHash, candidate)
			if err != nil {
				u.halt = true
				u.lastFail = err
				return nil, err
			}

			result.maturedBlocks = append(result.maturedBlocks, candidate)
			log.Printf("Mature block %v with %v tx, hash: %v", candidate.Height, len(blockByHash.TxsList), candidate.Hash[0:10], util.FormatReward(candidate.Reward))
      log.Println("Mature block key: ", candidate.RedisKey())
			break
		} else if hashFound { // it is an uncle
      orphan = false
      result.uncles++

      err := handleUncle(height, blockByHash, candidate)
      if err != nil {
        u.halt = true
        u.lastFail = err
        return nil, err
      }
      result.maturedBlocks = append(result.maturedBlocks, candidate)
      log.Printf("Mature uncle %v/%v of reward %v with hash: %v", candidate.Height, candidate.UncleHeight, util.FormatReward(candidate.Reward), blockByHash.Hash[0:10])
      log.Println("Mature uncle key: ", candidate.RedisKey())
    } 

		// Block is lost, we didn't find any valid block or uncle matching our data in a blockchain
		if orphan {
			result.orphans++
			candidate.Orphan = true
			result.orphanedBlocks = append(result.orphanedBlocks, candidate)
			log.Printf("Orphaned block %v:%v/%v", candidate.RoundHeight, candidate.Hash, candidate.Nonce)
      log.Println("Orphans block key: ", candidate.RedisKey())
		}
  }
	return result, nil
}

func matchCandidateByPreviousHash(nextBlock *rpc.BcBlockReply, candidate *storage.BlockData) bool {
  if nextBlock == nil {
    return false
  }
  // Just compare hash to previous hash of next block if block is unlocked as immature
  if ( len(candidate.Hash) > 0 && strings.EqualFold(candidate.Hash, nextBlock.PreviousHash) ) {
    return true
  }
   return false
}

func matchCandidate(block *rpc.BcBlockReply, candidate *storage.BlockData) bool {
  if block == nil {
    return false
  }
	// Just compare hash and nonce if block is unlocked as immature
	if ( len(candidate.Hash) > 0 && strings.EqualFold(candidate.Hash, block.Hash) &&
       len(candidate.Nonce) > 0 && strings.EqualFold(candidate.Nonce, block.Nonce) ) {
		return true
	}

	return false
}

func (u *BlockUnlocker) handleBlock(block *rpc.BcBlockReply, candidate *storage.BlockData) error {
	correctHeight := int64(block.Height)
	candidate.Height = correctHeight
	reward := getConstReward(block, candidate.Height)

	// Add TX fees
  /*
	extraTxReward, err := u.getExtraRewardForTx(block)
	if err != nil {
		return fmt.Errorf("Error while fetching TX receipt: %v", err)
	}
	if u.config.KeepTxFees {
		candidate.ExtraReward = extraTxReward
	} else {
		reward.Add(reward, extraTxReward)
	}
  */

  /* Bc handles uncles very differently
	// Add reward for including uncles
	uncleReward := getRewardForUncle(candidate.Height)
	rewardForUncles := big.NewInt(0).Mul(uncleReward, big.NewInt(int64(len(block.Uncles))))
	reward.Add(reward, rewardForUncles)
  */

	candidate.Orphan = false
	candidate.Hash = block.Hash
  candidate.Nonce = block.Nonce
	candidate.Reward = reward
	return nil
}

func handleUncle(height int64, uncle *rpc.BcBlockReply, candidate *storage.BlockData) error {
	uncleHeight := int64(uncle.Height)
	reward := getUncleReward(uncleHeight, height)
	candidate.Height = height
	candidate.UncleHeight = uncleHeight
	candidate.Orphan = false
	candidate.Hash = uncle.Hash
	candidate.Reward = reward
	return nil
}

func (u *BlockUnlocker) unlockPendingBlocks() {
	if u.halt {
		log.Println("Unlocking suspended due to last critical error:", u.lastFail)
		return
	}

	current, err := u.rpc.GetLatestBlock()
	if err != nil {
		//u.halt = true DUHHHHH... Ditto
		//u.lastFail = err
		log.Printf("Unable to get current blockchain height from node: Retrying")
		return
	}
	currentHeight, err := strconv.ParseInt(string(current.Number), 10, 64)
	if err != nil {
		u.halt = true
		u.lastFail = err
		log.Printf("Can't parse pending block number: %v", err)
		return
	}

	candidates, err := u.backend.GetCandidates(currentHeight - u.config.ImmatureDepth)
	if err != nil {
		u.halt = true
		u.lastFail = err
		log.Printf("Failed to get block candidates from backend: %v", err)
		return
	}

	if len(candidates) == 0 {
		log.Println("No block candidates to unlock")
		return
	}

	result, err := u.unlockCandidates(candidates)
	if err != nil {
		u.halt = true
		u.lastFail = err
		log.Printf("Failed to unlock blocks: %v", err)
		return
	}
	log.Printf("Immature %v blocks, %v uncles, %v orphans", result.blocks, result.uncles, result.orphans)

	err = u.backend.WritePendingOrphans(result.orphanedBlocks)
	if err != nil {
		u.halt = true
		u.lastFail = err
		log.Printf("Failed to insert orphaned blocks into backend: %v", err)
		return
	} else {
		log.Printf("Inserted %v orphaned blocks to backend", result.orphans)
	}

	totalRevenue := new(big.Rat)
	totalMinersProfit := new(big.Rat)
	totalPoolProfit := new(big.Rat)

	for _, block := range result.maturedBlocks {
		revenue, minersProfit, poolProfit, roundRewards, err := u.calculateRewards(block)
		if err != nil {
			u.halt = true
			u.lastFail = err
			log.Printf("Failed to calculate rewards for round %v: %v", block.RoundKey(), err)
			return
		}
		err = u.backend.WriteImmatureBlock(block, roundRewards)
		if err != nil {
			u.halt = true
			u.lastFail = err
			log.Printf("Failed to credit rewards for round %v: %v", block.RoundKey(), err)
			return
		}
		totalRevenue.Add(totalRevenue, revenue)
		totalMinersProfit.Add(totalMinersProfit, minersProfit)
		totalPoolProfit.Add(totalPoolProfit, poolProfit)

		logEntry := fmt.Sprintf(
			"IMMATURE %v: revenue %v, miners profit %v, pool profit: %v",
			block.RoundKey(),
			util.FormatRatReward(revenue),
			util.FormatRatReward(minersProfit),
			util.FormatRatReward(poolProfit),
		)
		entries := []string{logEntry}
		for login, reward := range roundRewards {
			entries = append(entries, fmt.Sprintf("\tREWARD %v: %v: %v Shannon", block.RoundKey(), login, reward))
		}
		log.Println(strings.Join(entries, "\n"))
	}

	log.Printf(
		"IMMATURE SESSION: revenue %v, miners profit %v, pool profit: %v",
		util.FormatRatReward(totalRevenue),
		util.FormatRatReward(totalMinersProfit),
		util.FormatRatReward(totalPoolProfit),
	)
}

func (u *BlockUnlocker) unlockAndCreditMiners() {
	if u.halt {
		log.Println("Unlocking suspended due to last critical error:", u.lastFail)
		return
	}

	current, err := u.rpc.GetLatestBlock()
	if err != nil {
		//u.halt = true #BCnode after 1.4.9 is weird. They do EOF when they sync to a fork, instead of just delaying
		//u.lastFail = err
		log.Printf("Unable to get current blockchain height from node: Waiting")
		return
	}
	currentHeight, err := strconv.ParseInt(string(current.Number), 10, 64)
	if err != nil {
		u.halt = true
		u.lastFail = err
		log.Printf("Can't parse pending block number: %v", err)
		return
	}

	immature, err := u.backend.GetImmatureBlocks(currentHeight - u.config.Depth)
	if err != nil {
		u.halt = true
		u.lastFail = err
		log.Printf("Failed to get block candidates from backend: %v", err)
		return
	}

	if len(immature) == 0 {
		log.Println("No immature blocks to credit miners")
		return
	}

	result, err := u.unlockCandidates(immature)
	if err != nil {
		u.halt = true
		u.lastFail = err
		log.Printf("Failed to unlock blocks: %v", err)
		return
	}
	log.Printf("Unlocked %v blocks, %v uncles, %v orphans", result.blocks, result.uncles, result.orphans)

	for _, block := range result.orphanedBlocks {
		err = u.backend.WriteOrphan(block)
		if err != nil {
			u.halt = true
			u.lastFail = err
			log.Printf("Failed to insert orphaned block into backend: %v", err)
			return
		}
	}
	log.Printf("Inserted %v orphaned blocks to backend", result.orphans)

	totalRevenue := new(big.Rat)
	totalMinersProfit := new(big.Rat)
	totalPoolProfit := new(big.Rat)

	for _, block := range result.maturedBlocks {
		revenue, minersProfit, poolProfit, roundRewards, err := u.calculateRewards(block)
		if err != nil {
			u.halt = true
			u.lastFail = err
			log.Printf("Failed to calculate rewards for round %v: %v", block.RoundKey(), err)
			return
		}
		err = u.backend.WriteMaturedBlock(block, roundRewards)
		if err != nil {
			u.halt = true
			u.lastFail = err
			log.Printf("Failed to credit rewards for round %v: %v", block.RoundKey(), err)
			return
		}
		totalRevenue.Add(totalRevenue, revenue)
		totalMinersProfit.Add(totalMinersProfit, minersProfit)
		totalPoolProfit.Add(totalPoolProfit, poolProfit)

		logEntry := fmt.Sprintf(
			"MATURED %v: revenue %v, miners profit %v, pool profit: %v",
			block.RoundKey(),
			util.FormatRatReward(revenue),
			util.FormatRatReward(minersProfit),
			util.FormatRatReward(poolProfit),
		)
		entries := []string{logEntry}
		for login, reward := range roundRewards {
			entries = append(entries, fmt.Sprintf("\tREWARD %v: %v: %v Shannon", block.RoundKey(), login, reward))
		}
		log.Println(strings.Join(entries, "\n"))
	}

	log.Printf(
		"MATURE SESSION: revenue %v, miners profit %v, pool profit: %v",
		util.FormatRatReward(totalRevenue),
		util.FormatRatReward(totalMinersProfit),
		util.FormatRatReward(totalPoolProfit),
	)
}

func (u *BlockUnlocker) calculateRewards(block *storage.BlockData) (*big.Rat, *big.Rat, *big.Rat, map[string]int64, error) {
	revenue := new(big.Rat).SetInt(block.Reward)
	minersProfit, poolProfit := chargeFee(revenue, u.config.PoolFee)

	shares, err := u.backend.GetRoundShares(block.RoundHeight, block.Hash)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	rewards := calculateRewardsForShares(shares, block.TotalShares, minersProfit)

	if block.ExtraReward != nil {
		extraReward := new(big.Rat).SetInt(block.ExtraReward)
		poolProfit.Add(poolProfit, extraReward)
		revenue.Add(revenue, extraReward)
	}

	if u.config.Donate {
		var donation = new(big.Rat)
		poolProfit, donation = chargeFee(poolProfit, donationFee)
		login := strings.ToLower(donationAccount)
		rewards[login] += weiToShannonInt64(donation)
	}

  poolFeeAddress := os.Getenv(u.config.PoolFeeAddress)
	if len(poolFeeAddress) != 0 {
		address := strings.ToLower(poolFeeAddress)
		rewards[address] += weiToShannonInt64(poolProfit)
	}
	log.Println("resulting rewards: ", revenue, minersProfit, poolProfit, rewards)
	return revenue, minersProfit, poolProfit, rewards, nil
}

func calculateRewardsForShares(shares map[string]int64, total int64, reward *big.Rat) map[string]int64 {
	rewards := make(map[string]int64)

	for login, n := range shares {
		percent := big.NewRat(n, total)
		workerReward := new(big.Rat).Mul(reward, percent)
		rewards[login] += weiToShannonInt64(workerReward)
	}
	return rewards
}

// Returns new value after fee deduction and fee value.
func chargeFee(value *big.Rat, fee float64) (*big.Rat, *big.Rat) {
	feePercent := new(big.Rat).SetFloat64(fee / 100)
	feeValue := new(big.Rat).Mul(value, feePercent)
	return new(big.Rat).Sub(value, feeValue), feeValue
}

func weiToShannonInt64(wei *big.Rat) int64 {
	shannon := new(big.Rat).SetInt(util.Shannon)
	inShannon := new(big.Rat).Quo(wei, shannon)
	value, _ := strconv.ParseInt(inShannon.FloatString(0), 10, 64)
	return value
}

func getConstReward(block *rpc.BcBlockReply, height int64) *big.Int {
	return new(big.Int).Mul(new(big.Int).Set(afterTargetReward), new(big.Int).SetInt64(int64(block.NrgGrant)))
}

func getRewardForUncle(height int64) *big.Int {
	//reward := getConstReward(height)
	return new(big.Int).SetInt64(0) //new(big.Int).Div(reward, new(big.Int).SetInt64(32))
}

func getUncleReward(uHeight, height int64) *big.Int {
	//reward := getConstReward(height)
	//k := height - uHeight
	//reward.Mul(big.NewInt(8-k), reward)
	//reward.Div(reward, big.NewInt(8))
	return new(big.Int).SetInt64(0) //reward
}

func (u *BlockUnlocker) getExtraRewardForTx(block *rpc.BcBlockReply) (*big.Int, error) {
	amount := new(big.Int)

	for _, tx := range block.TxsList {
		receipt, err := u.rpc.GetTxReceipt(tx.Hash)
		if err != nil {
			return nil, err
		}
		if receipt != nil {
      /* FIX ME - BC Transactions....
			gasUsed := util.String2Big(receipt.GasUsed)
			gasPrice := util.String2Big(tx.GasPrice)
      */
			fee := new(big.Int) //gasUsed, gasPrice)
			amount.Add(amount, fee)
		}
	}
	return amount, nil
}
