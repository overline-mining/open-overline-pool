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
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/lgray/open-overline-pool/util"
)

type RPCClient struct {
	sync.RWMutex
	Url         string
	Name        string
  SCookie     string
	sick        bool
	sickRate    int
	successRate int
	client      *http.Client
}

type GetBlockReply struct {
	Number       string   `json:"height"`
	Hash         string   `json:"hash"`
	Nonce        string   `json:"nonce"`
  Distance     string   `json:"distance"`
	Miner        string   `json:"miner"`
	Difficulty   string   `json:"difficulty"`
	GasLimit     string   
	GasUsed      string   
	Transactions []Tx     `json:"txsList"`
	Uncles       []string 
	// https://github.com/ethereum/EIPs/issues/95
	SealFields []string 
}

type GetBlockReplyPart struct {
	Number     string `json:"number"`
	Difficulty string `json:"difficulty"`
}

const receiptStatusSuccessful = "0x1"

type TxReceipt struct {
	TxHash    string `json:"hash"`
	GasUsed   string `json:"overline"`
	BlockHash string 
	Status    string `json:"nonce"`
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
	Gas      string
	GasPrice string 
	Hash     string `json:"hash"`
  Nonce    string `json:"nonce"`
}

type JSONRpcResp struct {
	Id     *json.RawMessage       `json:"id"`
	Result *json.RawMessage       `json:"result"`
	Error  map[string]interface{} `json:"error"`
}

func NewRPCClient(name, url, scookie, timeout string) *RPCClient {
	rpcClient := &RPCClient{Name: name, Url: url, SCookie: scookie}
	timeoutIntv := util.MustParseDuration(timeout)
	rpcClient.client = &http.Client{
		Timeout: timeoutIntv,
	}
	return rpcClient
}

func (r *RPCClient) GetWork() ([]string, error) {
	rpcResp, err := r.doPost(r.Url, "ol_getWork", []string{})
	if err != nil {
		return nil, err
	}
	var reply []string
	err = json.Unmarshal(*rpcResp.Result, &reply)
	return reply, err
}

func (r *RPCClient) GetLatestBlock() (*GetBlockReplyPart, error) {
	rpcResp, err := r.doPost(r.Url, "getLatestBlock", []string{})
	if err != nil {
		return nil, err
	}
	if rpcResp.Result != nil {
		var reply *GetBlockReplyPart
		err = json.Unmarshal(*rpcResp.Result, &reply)
		return reply, err
	}
	return nil, nil
}

func (r *RPCClient) GetBlockByHeight(height int64) (*GetBlockReply, error) {
	params := []interface{}{fmt.Sprintf("0x%x", height), true}
	return r.getBlockBy("getBlockHeight", params)
}

func (r *RPCClient) GetBlockByHash(hash string) (*GetBlockReply, error) {
	params := []interface{}{hash, true}
	return r.getBlockBy("getBlockHash", params)
}

func (r *RPCClient) GetUncleByBlockNumberAndIndex(height int64, index int) (*GetBlockReply, error) {
	params := []interface{}{fmt.Sprintf("0x%x", height), fmt.Sprintf("0x%x", index)}
	return r.getBlockBy("ol_getUncleByBlockNumberAndIndex", params)
}

func (r *RPCClient) getBlockBy(method string, params []interface{}) (*GetBlockReply, error) {
	rpcResp, err := r.doPost(r.Url, method, params)
	if err != nil {
		return nil, err
	}
	if rpcResp.Result != nil {
		var reply *GetBlockReply
		err = json.Unmarshal(*rpcResp.Result, &reply)
		return reply, err
	}
	return nil, nil
}

func (r *RPCClient) GetTxReceipt(hash string) (*TxReceipt, error) {
	rpcResp, err := r.doPost(r.Url, "getTx", []string{hash})
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
	rpcResp, err := r.doPost(r.Url, "ol_submitWork", params)
	if err != nil {
		return false, err
	}
	var reply bool
	err = json.Unmarshal(*rpcResp.Result, &reply)
	return reply, err
}

type BalanceReply struct {
  Confirmed string `json:"confirmed"`
  Unconfirmed string `json:"unconfirmed"`
  Collateralized string `json:"collateralized"`
  Unlockable string `json:"unlockable"`
}

func (r *RPCClient) GetBalance(address string) (*big.Int, error) {
	rpcResp, err := r.doPost(r.Url, "getBalance", []string{address})
	if err != nil {
		return nil, err
	}
	var reply BalanceReply
	err = json.Unmarshal(*rpcResp.Result, &reply)
	if err != nil {
		return nil, err
	}
	return util.String2Big(reply.Confirmed), err
}

func (r *RPCClient) Sign(from string, s string) (string, error) {
	hash := sha256.Sum256([]byte(s))
	rpcResp, err := r.doPost(r.Url, "ol_sign", []string{from, hexutil.Encode(hash[:])})
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

func (r *RPCClient) GetPeerCount() (int64, error) {
	rpcResp, err := r.doPost(r.Url, "net_peerCount", nil)
	if err != nil {
		return 0, err
	}
	var reply string
	err = json.Unmarshal(*rpcResp.Result, &reply)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.Replace(reply, "0x", "", -1), 16, 64)
}

func (r *RPCClient) SendTransaction(from, to, gas, gasPrice, value string, autoGas bool) (string, error) {
	params := map[string]string{
		"from":  from,
		"to":    to,
		"value": value,
	}
	if !autoGas {
		params["gas"] = gas
		params["gasPrice"] = gasPrice
	}
	rpcResp, err := r.doPost(r.Url, "sendTx", []interface{}{params})
	var reply string
	if err != nil {
		return reply, err
	}
	err = json.Unmarshal(*rpcResp.Result, &reply)
	if err != nil {
		return reply, err
	}
	/* There is an inconsistence in a "standard". Geth returns error if it can't unlock signer account,
	 * but Parity returns zero hash 0x000... if it can't send tx, so we must handle this case.
	 * https://github.com/ethereum/wiki/wiki/JSON-RPC#returns-22
	 */
	if util.IsZeroHash(reply) {
		err = errors.New("transaction is not yet available")
	}
	return reply, err
}

func (r *RPCClient) doPost(url string, method string, params interface{}) (*JSONRpcResp, error) {
	jsonReq := map[string]interface{}{"jsonrpc": "2.0", "method": method, "params": params, "id": 0}
	data, _ := json.Marshal(jsonReq)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
  req.SetBasicAuth("", r.SCookie)
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
	_, err := r.GetWork()
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
