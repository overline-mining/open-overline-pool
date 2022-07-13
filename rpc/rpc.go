package rpc

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/lgray/open-overline-pool/util"
)

type BcRpcError struct {
	Details string `json:"details"`
}

type BcTransactionResponse struct {
	Status uint64 `json:"status"`
	TxHash string `json:"txHash"`
	Error  string `json:"error"`
}

type BcTxOutPoint struct {
	Value string `json:"value"`
	Hash  string `json:"hash"`
	Index int64  `json:"index"`
}

type BcTxInput struct {
	OutPoint     BcTxOutPoint `json:"outPoint"`
	ScriptLength uint64       `json:"script_Length"`
	InputScript  string       `json:"input_Script"`
}

type BcTxOutput struct {
	Value        string `json:"value"`
	Unit         string `json:"unit"`
	ScriptLength uint64 `json:"script_Length"`
	OutputScript string `json:"output_Script"`
}

type BcTransaction struct {
	Version     uint64       `json:"version"`
	Nonce       string       `json:"nonce"`
	Hash        string       `json:"hash"`
	Overline    string       `json:"overline"`
	NinCount    uint64       `json:"nin_count"`
	NoutCount   uint64       `json:"nout_Count"`
	InputsList  []BcTxInput  `json:"inputsList"`
	OutputsList []BcTxOutput `json:"outputsList"`
	LockTime    uint64       `json:"lockTime"`
}

type BcChildBlockHeader struct {
	Blockchain                           string   `json:"blockchain"`
	Hash                                 string   `json:"hash"`
	PreviousHash                         string   `json:"previousHash"`
	Timestamp                            uint64   `json:"timestamp"`
	Height                               uint64   `json:"height"`
	MerkleRoot                           string   `json:"merkleRoot"`
	BlockchainConfirmationsInParentCount uint64   `json:"blockchainConfirmationsInParentCount"`
	MarkedTxsList                        []string `json:"-"` // fill out later!
	MarkedTxCount                        uint64   `json:"markedTxCount"`
}

type BcChildBlockHeaders struct {
	BtcList                    []BcChildBlockHeader `json:"btcList"`
	EthList                    []BcChildBlockHeader `json:"ethList"`
	LskList                    []BcChildBlockHeader `json:"lskList"`
	NeoList                    []BcChildBlockHeader `json:"neoList"`
	WavList                    []BcChildBlockHeader `json:"wavList"`
	BlockchainFingerprintsRoot string               `json:"blockchainFingerprintsRoot"`
}

type BcBlockReply struct {
	Hash                       string              `json:"hash"`
	PreviousHash               string              `json:"previous_hash"`
	Version                    uint64              `json:"version"`
	SchemaVersion              uint64              `json:"schema_version"`
	Height                     uint64              `json:"height"`
	Difficulty                 string              `json:"difficulty"`
	Timestamp                  uint64              `json:"timestamp"`
	MerkleRoot                 string              `json:"merkle_root"`
	ChainRoot                  string              `json:"chain_root"`
	Distance                   string              `json:"distance"`
	TotalDistance              string              `json:"total_distance"`
	Nonce                      string              `json:"nonce"`
	NrgGrant                   float64             `json:"nrg_grant"`
	EmblemWeight               float64             `json:"emblem_weight"`
	EmblemChainFingerprintRoot string              `json:"emblem_chain_fingerprint_Root"`
	EmblemChainAddress         string              `json:"emblem_chain_address"`
	TxCount                    uint64              `json:"tx_Count"`
	TxsList                    []BcTransaction     `json:"txs"`
	TxFeeBase                  uint64              `json:"txFeeBase"`
	TxDistanceSumLimit         uint64              `json:"txDistanceSumLimit"`
	BlockchainHeadersCount     uint64              `json:"blockchain_headers_count"`
	BlockchainHeaders          BcChildBlockHeaders `json:"blockchain_headers"`
}

type RPCClient struct {
	sync.RWMutex
	Url         string
	Urlgool     string
	Name        string
	SCookie     string
	sick        bool
	sickRate    int
	successRate int
	client      *http.Client
}

type GetBlockReply struct {
	Number       json.Number `json:"height,Number"`
	Hash         string      `json:"hash"`
	Nonce        string      `json:"nonce"`
	Distance     string      `json:"distance"`
	Miner        string      `json:"miner"`
	Difficulty   string      `json:"difficulty"`
	GasLimit     string
	GasUsed      string
	Transactions []Tx `json:"txsList"`
	Uncles       []string
	// https://github.com/ethereum/EIPs/issues/95
	SealFields []string
}

type GetBlockReplyPart struct {
	Number     json.Number `json:"height"`
	Difficulty string      `json:"difficulty"`
	Hash       string      `json:"hash"`
	Distance   string      `json:"distance"`
}

const receiptStatusSuccessful = "0x1"

type TxReceipt struct {
	TxHash    string `json:"hash"`
	GasUsed   string `json:"overline"`
	Nonce     string `json:"nonce"`
	BlockHash string
	Status    string `json:"nonce"`
}

func (r *TxReceipt) Confirmed() bool {
	return len(r.BlockHash) > 0
}

// Use with previous method
func (r *TxReceipt) Successful() bool {
	return len(r.TxHash) > 0
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

func NewRPCClient(name, url, urlgool, scookie, timeout string) *RPCClient {
	rpcClient := &RPCClient{Name: name, Url: url, SCookie: scookie, Urlgool: urlgool}
	timeoutIntv := util.MustParseDuration(timeout)
	rpcClient.client = &http.Client{
		Timeout: timeoutIntv,
	}
	return rpcClient
}

func (r *RPCClient) GetWork() ([]string, error) {
	rpcResp, err := r.doPost(r.Url, "ol_getWork", []string{}) // fixme!
	if err != nil {
		return nil, err
	}
	var reply []string
	err = json.Unmarshal(*rpcResp.Result, &reply)
	return reply, err
}

func (r *RPCClient) GetLatestBlock() (*GetBlockReplyPart, error) {
	rpcResp, err := r.doPost(r.Urlgool, "ovl_getHighestBlock", []string{})
	if err != nil {
		return nil, err
	}
	if rpcResp.Result != nil {
		var reply *GetBlockReplyPart
		err = json.Unmarshal(*rpcResp.Result, &reply)
		log.Println(reply)
		return reply, err
	}
	return nil, nil
}

func (r *RPCClient) GetBlockByHeight(height int64) (*BcBlockReply, error) {
	var params []interface{}
	params = append(params, height)
	return r.getNewBlockBy("ovl_getBlockByHeight", params)
}

func (r *RPCClient) GetBlockByHash(hash string) (*BcBlockReply, error) {
	params := []string{hash}
	return r.getBlockBy("ovl_getBlockByHash", params)
}

//unused if i see correctly
func (r *RPCClient) GetUncleByBlockNumberAndIndex(height int64, index int) (*BcBlockReply, error) {
	params := []string{strconv.FormatInt(height, 10), strconv.FormatInt(int64(index), 10)}
	return r.getBlockBy("ol_getUncleByBlockNumberAndIndex", params)
}

func (r *RPCClient) getBlockBy(method string, params []string) (*BcBlockReply, error) {

	rpcResp, err := r.doPost(r.Urlgool, method, params)
	log.Println("rpcresp")
	log.Println(rpcResp)
	log.Println("error")
	log.Println(err)
	log.Println("hereiam2.1")
	var reply *BcBlockReply
	if err == nil {
		err = json.Unmarshal(*rpcResp.Result, &reply)
	}
	log.Println("hereiam3")
	return reply, err
}

func (r *RPCClient) getNewBlockBy(method string, params []interface{}) (*BcBlockReply, error) {

	rpcResp, err := r.doNewPost(r.Urlgool, method, params)
	log.Println("rpcresp")
	log.Println(rpcResp)
	log.Println("error")
	log.Println(err)
	//if err != nil {
	//	return nil, err
	//}

	log.Println("hereiam2")
	var reply *BcBlockReply
	if err == nil {
		err = json.Unmarshal(*rpcResp.Result, &reply)
	}
	log.Println("hereiam3")
	return reply, err

}

func (r *RPCClient) GetTxReceipt(hash string) (*TxReceipt, error) {
	rpcResp, err := r.doPost(r.Urlgool, "ovl_getBlockByTx", []string{hash})
	if err != nil {
		return nil, err
	}
	if rpcResp.Result != nil {
		var reply *BcBlockReply
		err = json.Unmarshal(*rpcResp.Result, &reply)
		out := new(TxReceipt)
		out.BlockHash = reply.Hash
		out.TxHash = hash
		log.Println("got rpc back from node (its scary)", out.BlockHash, out.TxHash, err)
		return out, err
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
	Confirmed      string `json:"confirmed"`
	Unconfirmed    string `json:"unconfirmed"`
	Collateralized string `json:"collateralized"`
	Unlockable     string `json:"unlockable"`
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
	amountInNRG := util.String2Big(reply.Confirmed)
	amountInWei := new(big.Int).Mul(amountInNRG, util.Ether)
	return amountInWei, err
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

func (r *RPCClient) SendTransaction(from, to, valueInWei, pkey string) (string, error) {
	etherString := util.Ether.String()
	valueInNRG, _ := new(big.Rat).SetString(valueInWei + "/" + etherString)
	log.Println("constructed value in NRG -> ", valueInNRG.FloatString(18))
	params := []string{from, to, valueInNRG.FloatString(18), "0", pkey}

	rpcResp, err := r.doPost(r.Url, "newTx", params)
	var reply BcTransactionResponse
	if err != nil {
		return reply.Error, err
	}
	err = json.Unmarshal(*rpcResp.Result, &reply)
	if err != nil {
		return reply.Error, err
	}

	log.Println("tx reply -> ", reply.TxHash)

	if util.IsZeroHash(reply.TxHash) {
		err = errors.New("transaction is not yet available")
	}
	return reply.TxHash, err
}

func (r *RPCClient) doPost(url string, method string, params []string) (*JSONRpcResp, error) {
	jsonReq := map[string]interface{}{"jsonrpc": "2.0", "method": method, "params": params, "id": 0}
	data, _ := json.Marshal(jsonReq)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.SetBasicAuth("", "correct-horse-battery-staple")
	req.Header.Set("Content-Length", (string)(len(data)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if r.Name == "BlockUnlocker" {
		log.Println(r.Name)
		log.Println(jsonReq)
		log.Println(req)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		r.markSick()
		return nil, err
	}
	log.Println(resp)
	defer resp.Body.Close()

	var rpcResp *JSONRpcResp
	err = json.NewDecoder(resp.Body).Decode(&rpcResp)

	if err != nil {
		r.markSick()
		return nil, err
	}
	if rpcResp.Error != nil {
		//log.Println(rpcResp)
		r.markSick()
		return nil, errors.New(rpcResp.Error["message"].(string))
	}

	return rpcResp, err
}

func (r *RPCClient) doNewPost(url string, method string, params []interface{}) (*JSONRpcResp, error) {
	jsonReq := map[string]interface{}{"jsonrpc": "2.0", "method": method, "params": params, "id": 0}
	data, _ := json.Marshal(jsonReq)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.SetBasicAuth("", "correct-horse-battery-staple")
	req.Header.Set("Content-Length", (string)(len(data)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if r.Name == "BlockUnlocker" {
		log.Println(r.Name)
		log.Println(jsonReq)
		log.Println(req)
	}
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
		//log.Println(rpcResp)
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
