package rpc

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zano-mining/open-zano-pool/util"
)

type RPCClient struct {
	sync.RWMutex
	Url         string
	Name        string
	sick        bool
	sickRate    int
	successRate int
	client      *http.Client
}

type WorkRequestParams struct {
  ExtraText     string   `json:"extra_text"`
  WalletAddress string   `json:"wallet_address"`
  StakeAddress  string   `json:"stakeholder_address"`
  PosBlock      bool     `json:"pos_block"`
  PosAmount     int      `json:"pos_amount"`
  PosIndex      int      `json:"pos_index"`
}

type GetBlockTemplateReply struct {
  Blob         string   `json:"blocktemplate_blob"`
  Header       string   `json:"blocktemplate_work"`
  Difficulty   string   `json:"difficulty"`
  Height       uint64   `json:"height"`
  PrevHash     string   `json:"prev_hash"`
  Seed         string   `json:"seed"`
  Status       string   `json:"status"`
}

type GetBlockHeader struct {
  Depth        uint64 `json:"depth"`
  Difficulty   string `json:"difficulty"`
  Hash         string `json:"hash"`
  Height       uint64 `json:"height"`
  Nonce        uint64 `json:"nonce"`
  OrphanStatus bool   `json:"orphan_status"`
  Reward       uint64 `json:"reward"`
  Timestamp    uint64 `json:"timestamp"`
}

type GetBlockHeaderReply struct {
  BlockHeader GetBlockHeader `json:"block_header"`  
}
  
type GetBlockReply struct {
	Number       string   `json:"number"`
	Hash         string   `json:"hash"`
	Nonce        string   `json:"nonce"`
	Miner        string   `json:"miner"`
	Difficulty   string   `json:"difficulty"`
  Reward       uint64   `json:"reward"`
  OrphanStatus bool     `json:"orphan_status"`
	GasLimit     string   `json:"gasLimit"`
	GasUsed      string   `json:"gasUsed"`
	Transactions []Tx     `json:"transactions"`
	Uncles       []string `json:"uncles"`
	// https://github.com/ethereum/EIPs/issues/95
	SealFields []string `json:"sealFields"`
}

type GetBlockReplyHeaderPartRaw struct {
  Number     uint64 `json:"height"`
  Difficulty string `json:"difficulty"`
  Hash       string `json:"hash"`
}

type GetBlockReplyPart struct {
	Number     string `json:"number"`
	Difficulty string `json:"difficulty"`
  Hash       string `json:"hash"`
}

type GetBlockReplyPartRaw struct {
  BlockHeader GetBlockReplyHeaderPartRaw `json:"block_header"`
}

type SubmitBlockReply struct {
  Status string `json:"status"`
}

const receiptStatusSuccessful = "0x1"

type TransferDestination struct {
  Address string `json:"address"`
  Amount  uint64 `json:"amount"`
}

type Transfer struct {
  Destinations []TransferDestination `json:"destinations"`
  Fee          uint64 `json:"fee"`
  Mixin        uint64 `json:"mixin"`
}

type TransferReply struct {
  TxHash        string `json:"tx_hash"`
  TxUnsignedHex string `json:"tx_unsigned_hex"`
}
  
type GetInfoReply struct {
  OutgoingConnections uint64 `json:"outgoing_connections_count"`
}
    
type TxReceipt struct {
	TxHash    string `json:"transactionHash"`
	GasUsed   string `json:"gasUsed"`
	BlockHash string `json:"blockHash"`
	Status    string `json:"status"`
}

func (r *TxReceipt) Confirmed() bool {
	return len(r.BlockHash) > 0
}

// Use with previous method
func (r *TxReceipt) Successful() bool {
	if len(r.Status) > 0 {
		return r.Status == receiptStatusSuccessful
	}
	return true
}

type Tx struct {
	Gas      string `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Hash     string `json:"hash"`
}

type JSONRpcResp struct {
	Id     *json.RawMessage       `json:"id"`
	Result *json.RawMessage       `json:"result"`
	Error  map[string]interface{} `json:"error"`
}

func NewRPCClient(name, url, timeout string) *RPCClient {
	rpcClient := &RPCClient{Name: name, Url: url}
	timeoutIntv := util.MustParseDuration(timeout)
	rpcClient.client = &http.Client{
		Timeout: timeoutIntv,
	}
	return rpcClient
}

func (r *RPCClient) GetWork(miner_address string) ([]string, error) {
  var wparams WorkRequestParams
  wparams.ExtraText = "open-zano-pool"
  wparams.WalletAddress = miner_address
  wparams.StakeAddress = miner_address
  wparams.PosBlock = false
  wparams.PosAmount = 0
  wparams.PosIndex = 0
	rpcResp, err := r.doPost(r.Url, "getblocktemplate", wparams)
	if err != nil {
		return nil, err
	}
	var replyJson *GetBlockTemplateReply
	err = json.Unmarshal(*rpcResp.Result, &replyJson)

  reply := []string{
    replyJson.Header,
    "0x" + replyJson.Seed,
    util.GetTargetHexFromString(replyJson.Difficulty),
    util.ToHexUint(replyJson.Height),
    "0x" + replyJson.Blob,
  }
  
	return reply, err
}

func (r * RPCClient) VerifySolution(params []string) (*bool, error) {
  rpcResp, err := r.doPost(r.Url, "checksolution", params)
  if err != nil {
    return nil, err
  }
  if rpcResp.Result != nil {
    reply := new(bool)
    *reply, _ = strconv.ParseBool(string(*rpcResp.Result))
    if err != nil {
      return nil, err
    }
    return reply, err
  }
  return nil, nil
}

func (r *RPCClient) GetLatestBlock() (*GetBlockReplyPart, error) {
	rpcResp, err := r.doPost(r.Url, "getlastblockheader", []string{})
	if err != nil {
		return nil, err
	}
	if rpcResp.Result != nil {
		var replyRaw *GetBlockReplyPartRaw
		err = json.Unmarshal(*rpcResp.Result, &replyRaw)
    if err != nil {
      return nil, err
    }
    var reply = new(GetBlockReplyPart)
    reply.Number = util.ToHexUint(replyRaw.BlockHeader.Number)
    reply.Difficulty = replyRaw.BlockHeader.Difficulty
    reply.Hash = replyRaw.BlockHeader.Hash

		return reply, err
	}
	return nil, nil
}

func (r *RPCClient) GetBlockByHeight(height int64) (*GetBlockReply, error) {
	params := map[string]int64{"height": height}
	return r.getBlockBy("getblockheaderbyheight", params)
}

func (r *RPCClient) GetBlockByHash(hash string) (*GetBlockReply, error) {
	params := map[string]string{"hash": hash}
	return r.getBlockBy("getblockheaderbyhash", params)
}

func (r *RPCClient) GetUncleByBlockNumberAndIndex(height int64, index int) (*GetBlockReply, error) {
	params := []interface{}{fmt.Sprintf("0x%x", height), fmt.Sprintf("0x%x", index)}
	return r.getBlockBy("eth_getUncleByBlockNumberAndIndex", params)
}

func (r *RPCClient) getBlockBy(method string, params interface{}) (*GetBlockReply, error) {
	rpcResp, err := r.doPost(r.Url, method, params)
	if err != nil {
		return nil, err
	}
	if rpcResp.Result != nil {
		var reply *GetBlockHeaderReply
		err = json.Unmarshal(*rpcResp.Result, &reply)

    out := new(GetBlockReply)
    out.Number = util.ToHexUint(reply.BlockHeader.Height)
    out.Hash = "0x" + reply.BlockHeader.Hash
    out.Nonce = util.ToHexUintNoPad(reply.BlockHeader.Nonce)
    out.Miner = ""
    out.Difficulty = reply.BlockHeader.Difficulty
    out.Reward = reply.BlockHeader.Reward
    out.OrphanStatus = reply.BlockHeader.OrphanStatus
  	return out, err
	}
	return nil, nil
}

func (r *RPCClient) GetTxReceipt(hash string) (*TxReceipt, error) {
	rpcResp, err := r.doPost(r.Url, "eth_getTransactionReceipt", []string{hash})
	if err != nil {
		return nil, err
	}
	if rpcResp.Result != nil {
		var reply *TxReceipt
		err = json.Unmarshal(*rpcResp.Result, &reply)
		return reply, err
	}
	return nil, nil
}

func (r *RPCClient) SubmitBlock(params []string) (bool, error) {
	rpcResp, err := r.doPost(r.Url, "submitblock", params)
	if err != nil {
		return false, err
	}
	var reply *SubmitBlockReply
	err = json.Unmarshal(*rpcResp.Result, &reply)
  if reply.Status == "OK" {
    return true, err
  }
	return false, err
}

func (r *RPCClient) GetBalance() (*big.Int, error) {
	rpcResp, err := r.doPost(r.Url, "getbalance", nil)
	if err != nil {
		return nil, err
	}
	var reply map[string]uint64
	err = json.Unmarshal(*rpcResp.Result, &reply)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetUint64(reply["unlocked_balance"]), err
}

func (r *RPCClient) Sign(from string, s string) (string, error) {
	hash := sha256.Sum256([]byte(s))
	rpcResp, err := r.doPost(r.Url, "eth_sign", []string{from, hexutil.Encode(hash[:])})
	var reply string
	if err != nil {
		return reply, err
	}
	err = json.Unmarshal(*rpcResp.Result, &reply)
	if err != nil {
		return reply, err
	}
	if util.IsZeroHash(reply) {
		err = errors.New("Can't sign message, perhaps account is locked")
	}
	return reply, err
}

func (r *RPCClient) GetPeerCount() (uint64, error) {
	rpcResp, err := r.doPost(r.Url, "getinfo", nil)
	if err != nil {
		return 0, err
	}
	var reply GetInfoReply
	err = json.Unmarshal(*rpcResp.Result, &reply)
	if err != nil {
		return 0, err
	}
	return reply.OutgoingConnections, err
}

func (r *RPCClient) SendTransaction(destinations []TransferDestination, fee uint64, mixin uint64) (string, error) {
  var transfer Transfer
  transfer.Destinations = destinations
  transfer.Fee = fee
  transfer.Mixin = mixin
  rpcResp, err := r.doPost(r.Url, "transfer", transfer)
	var reply TransferReply
	if err != nil {
		return "0x0", err
	}
	err = json.Unmarshal(*rpcResp.Result, &reply)
	if err != nil {
		return "0x0", err
	}
	if util.IsZeroHash(reply.TxHash) {
		err = errors.New("transaction is not yet available")
	}
	return "0x"+reply.TxHash, err
}

func (r *RPCClient) doPost(url string, method string, params interface{}) (*JSONRpcResp, error) {
	jsonReq := map[string]interface{}{"jsonrpc": "2.0", "method": method, "params": params, "id": 0}
	data, _ := json.Marshal(jsonReq)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Length", (string)(len(data)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		r.markSick()
		return nil, err
	}
	defer resp.Body.Close()
  
	var rpcResp *JSONRpcResp
	err = json.NewDecoder(resp.Body).Decode(&rpcResp)
	if err != nil {
		r.markSick()
		return nil, err
	}
	if rpcResp.Error != nil {
		r.markSick()
		return nil, errors.New(rpcResp.Error["message"].(string))
	}
	return rpcResp, err
}

func (r *RPCClient) Check() bool {
	_, err := r.GetWork("0")
	if err != nil {
		return false
	}
	r.markAlive()
	return !r.Sick()
}

func (r *RPCClient) Sick() bool {
	r.RLock()
	defer r.RUnlock()
	return r.sick
}

func (r *RPCClient) markSick() {
	r.Lock()
	r.sickRate++
	r.successRate = 0
	if r.sickRate >= 5 {
		r.sick = true
	}
	r.Unlock()
}

func (r *RPCClient) markAlive() {
	r.Lock()
	r.successRate++
	if r.successRate >= 5 {
		r.sick = false
		r.sickRate = 0
		r.successRate = 0
	}
	r.Unlock()
}
