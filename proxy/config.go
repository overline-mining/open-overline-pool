package proxy

import (
	"github.com/lgray/open-overline-pool/api"
	"github.com/lgray/open-overline-pool/payouts"
	"github.com/lgray/open-overline-pool/policy"
	"github.com/lgray/open-overline-pool/storage"
)

type Config struct {
	Name                  string        `json:"name"`
	Proxy                 Proxy         `json:"proxy"`
	Api                   api.ApiConfig `json:"api"`
	Upstream              []Upstream    `json:"upstream"`
	UpstreamCheckInterval string        `json:"upstreamCheckInterval"`

	Threads int `json:"threads"`

	Coin  string         `json:"coin"`
	Redis storage.Config `json:"redis"`

	BlockUnlocker payouts.UnlockerConfig `json:"unlocker"`
	Payouts       payouts.PayoutsConfig  `json:"payouts"`

	NewrelicName    string `json:"newrelicName"`
	NewrelicKey     string `json:"newrelicKey"`
	NewrelicVerbose bool   `json:"newrelicVerbose"`
	NewrelicEnabled bool   `json:"newrelicEnabled"`
}

type Proxy struct {
	Enabled              bool    `json:"enabled"`
	Listen               string  `json:"listen"`
	LimitHeadersSize     int     `json:"limitHeadersSize"`
	LimitBodySize        int64   `json:"limitBodySize"`
	BehindReverseProxy   bool    `json:"behindReverseProxy"`
	BlockRefreshInterval string  `json:"blockRefreshInterval"`
	Difficulty           int64   `json:"difficulty"`
  Diff2HashRate        float64 `json:"diffToHashRate"`
	StateUpdateInterval  string  `json:"stateUpdateInterval"`
	HashrateExpiration   string  `json:"hashrateExpiration"`

	Policy policy.Config `json:"policy"`

	MaxFails    int64 `json:"maxFails"`
	HealthCheck bool  `json:"healthCheck"`

	Stratum Stratum `json:"stratum"`
}

type Stratum struct {
	Enabled  bool   `json:"enabled"`
	Listen   string `json:"listen"`
	Timeout  string `json:"timeout"`
	MaxConn  int    `json:"maxConn"`
	TLS      bool   `json:"tls"`
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
}

type Upstream struct {
	Name      string `json:"name"`
	Url       string `json:"url"`
	UrlMining string `json:"urlMining"`
        SCookie   string `json:"scookie"`
	Timeout   string `json:"timeout"`
}
