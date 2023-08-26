package clique

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
)

type BlsSignerFn func(bytes []byte) bls.Signature

type BuilderClient struct {
	hc      *http.Client
	baseURL *url.URL
	blsKey  bls.SecretKey
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

func NewBuilderClient(host string, timeout time.Duration, blsKey bls.SecretKey) (*BuilderClient, error) {
	u, err := urlForHost(host)
	if err != nil {
		return nil, err
	}

	hc := &http.Client{Timeout: timeout}
	return &BuilderClient{
		hc:      hc,
		baseURL: u,
		blsKey:  blsKey,
	}, nil
}

type ValidatorRegistration struct {
	FeeRecipient string `json:"fee_recipient"`
	GasLimit     string `json:"gas_limit"`
	Timestamp    string `json:"timestamp"`
	Pubkey       string `json:"pubkey"`
}

type SignedValidatorRegistration struct {
	Message   ValidatorRegistration `json:"message"`
	Signature string                `json:"signature"`
}

type ExecutionPayloadResponse struct {
	Version string                `json:"version"`
	Data    engine.ExecutableData `json:"data"`
}

type GetHeaderResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (res *ExecutionPayloadResponse) getBlock() (*types.Block, error) {
	return engine.ExecutableDataToBlock(res.Data, nil)
}

func (bc *BuilderClient) RegisterValidator(feeRecipient string, gasLimit uint64) error {
	url := bc.baseURL.JoinPath("/eth/v1/builder/validators")
	regMsg := ValidatorRegistration{
		FeeRecipient: feeRecipient,
		GasLimit:     strconv.FormatUint(gasLimit, 10),
		Timestamp:    strconv.FormatInt(time.Now().Unix(), 10),
		Pubkey:       hexutil.Encode(bc.blsKey.PublicKey().Marshal()),
	}

	msg, err := json.Marshal(regMsg)
	if err != nil {
		return err
	}

	signedReg := &SignedValidatorRegistration{
		Message:   regMsg,
		Signature: hexutil.Encode(bc.blsKey.Sign(msg).Marshal()),
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

func (bc *BuilderClient) GetHeader(slot uint64, parentHash common.Hash) error {
	part := fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", slot, parentHash.Hex(), hexutil.Encode(bc.blsKey.PublicKey().Marshal()))
	url := bc.baseURL.JoinPath(part)
	resp, err := bc.hc.Get(url.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var response GetHeaderResponse
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&response)
	if err != nil {
		return err
	}
	if response.Code != 200 {
		return errors.New(response.Message)
	}
	return nil
}

func (bc *BuilderClient) GetBlock(slot uint64, parentHash common.Hash) (*types.Block, error) {
	// /eth/v1/builder/block/:slot/:parent_hash/:pubkey
	part := fmt.Sprintf("/eth/v1/builder/block/%d/%s/%s", slot, parentHash.Hex(), hexutil.Encode(bc.blsKey.PublicKey().Marshal()))
	url := bc.baseURL.JoinPath(part)
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
