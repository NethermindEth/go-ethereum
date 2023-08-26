package clique

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

type BuilderClient struct {
	hc      *http.Client
	baseURL *url.URL
}

func urlForHost(h string) (*url.URL, error) {
	// try to parse as url (being permissive)
	u, err := url.Parse(h)
	if err == nil && u.Host != "" {
		return u, nil
	}
	// try to parse as host:port
	host, port, err := net.SplitHostPort(h)
	if err != nil {
		return nil, errors.New("hostname must include port, separated by one colon, like example.com:3500")
	}
	return &url.URL{Host: net.JoinHostPort(host, port), Scheme: "http"}, nil
}

func NewBuilderClient(host string, timeout time.Duration) (*BuilderClient, error) {
	u, err := urlForHost(host)
	if err != nil {
		return nil, err
	}

	hc := &http.Client{Timeout: timeout}
	return &BuilderClient{
		hc:      hc,
		baseURL: u,
	}, nil
}

type ValidatorRegistration struct {
	FeeRecipient hexutil.Bytes `json:"fee_recipient"`
	GasLimit     string        `json:"gas_limit"`
	Timestamp    string        `json:"timestamp"`
	Pubkey       hexutil.Bytes `json:"pubkey"`
}

type SignedValidatorRegistration struct {
	Message   ValidatorRegistration `json:"message"`
	Signature hexutil.Bytes         `json:"signature"`
}

type ExecutionPayloadResponse struct {
	Version string                `json:"version"`
	Data    engine.ExecutableData `json:"data"`
}

func (res *ExecutionPayloadResponse) getBlock() (*types.Block, error) {
	return engine.ExecutableDataToBlock(res.Data, nil)
}

func (bc *BuilderClient) RegisterValidator(secretKey bls.SecretKey, feeRecipient hexutil.Bytes, gasLimit, timestamp string) error {
	url := bc.baseURL.JoinPath("/eth/v1/builder/validators")
	regMsg := ValidatorRegistration{
		FeeRecipient: feeRecipient,
		GasLimit:     gasLimit,
		Timestamp:    timestamp,
		Pubkey:       secretKey.PublicKey().Marshal(),
	}
	msg, err := json.Marshal(regMsg)
	if err != nil {
		return err
	}

	signedReg := &SignedValidatorRegistration{
		Message:   regMsg,
		Signature: secretKey.Sign(msg).Marshal(),
	}

	body, err := json.Marshal(signedReg)
	if err != nil {
		return err
	}

	_, err = bc.hc.Post(url.String(), "application/json", bytes.NewBuffer(body))

	if err != nil {
		return err
	}
	return nil
}

func (bc *BuilderClient) GetBlock(slot uint64, parentHash common.Hash, pubKey bls.PublicKey) (*types.Block, error) {
	// /eth/v1/builder/block/:slot/:parent_hash/:pubkey
	url := bc.baseURL.JoinPath("/eth/v1/builder/block",
		strconv.FormatUint(slot, 10),
		parentHash.Hex(),
		common.Bytes2Hex(pubKey.Marshal()))
	resp, err := bc.hc.Get(url.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var response ExecutionPayloadResponse
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&response)
	if err != nil {
		return nil, err
	}
	return response.getBlock()
}
