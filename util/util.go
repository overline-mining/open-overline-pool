package util

import (
	"math/big"
	"regexp"
	"strconv"
	"time"

  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
)

var Ether = math.BigPow(10, 12)
var Shannon = math.BigPow(10, 6)

var pow256 = math.BigPow(2, 256)
var u256max = new(big.Int).Sub(pow256, new(big.Int).SetInt64(1))
var addressPattern = regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
var addressPattern2 = regexp.MustCompile("[0-9a-zA-Z]{97}")
var zeroHash = regexp.MustCompile("^0?x?0+$")

func IsValidHexAddress(s string) bool {
	if IsZeroHash(s) || !addressPattern.MatchString(s) {
		return false
	}
	return true
}

func IsValidZanoAddress(s string) bool {
  if IsZeroHash(s) || !addressPattern2.MatchString(s) {
    return false
  }
  return true
}

func IsZeroHash(s string) bool {
	return zeroHash.MatchString(s)
}

func MakeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func GetTargetHexFromString(diff string) string {
  difficulty, _ := new(big.Int).SetString(diff, 10)
  diff1 := new(big.Int).Div(u256max, difficulty)
  return string(hexutil.Encode(diff1.Bytes()))
}
  
func GetTargetHex(diff int64) string {
	difficulty := big.NewInt(diff)
	diff1 := new(big.Int).Div(u256max, difficulty)
	return string(hexutil.Encode(diff1.Bytes()))
}

func TargetHexToDiff(targetHex string) *big.Int {
	targetBytes := common.FromHex(targetHex)
	return new(big.Int).Div(u256max, new(big.Int).SetBytes(targetBytes))
}

func ToHex(n int64) string {
	return "0x0" + strconv.FormatInt(n, 16)
}

func ToHexUint(n uint64) string {
  return "0x0" + strconv.FormatUint(n, 16)
}

func ToHexUintNoPad(n uint64) string {
  return "0x" + strconv.FormatUint(n, 16)
}

func FormatReward(reward *big.Int) string {
	return reward.String()
}

func FormatRatReward(reward *big.Rat) string {
	wei := new(big.Rat).SetInt(Ether)
	reward = reward.Quo(reward, wei)
	return reward.FloatString(8)
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func MustParseDuration(s string) time.Duration {
	value, err := time.ParseDuration(s)
	if err != nil {
		panic("util: Can't parse duration `" + s + "`: " + err.Error())
	}
	return value
}

func String2Big(num string) *big.Int {
	n := new(big.Int)
	n.SetString(num, 0)
	return n
}
