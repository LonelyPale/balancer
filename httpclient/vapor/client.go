package vapor

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/vapor/consensus"
	vprbc "github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"

	"github.com/bytom/blockcenter/balancer"
	"github.com/bytom/blockcenter/balancer/httpclient"
	vpr "github.com/bytom/blockcenter/coin/vapor"
	"github.com/bytom/blockcenter/protocol"
	"github.com/bytom/blockcenter/protocol/vapor"
	"github.com/bytom/blockcenter/service"
)

var vaporErrMap = map[string]error{
	"BTM716": service.ErrInputUTXONotFound,
}

type Client struct {
	httpclient.HttpClient
	NetParam string
	errMap   map[string]error
}

func NewClient(opts balancer.Options) (*Client, error) {
	client, err := httpclient.New(opts)
	if err != nil {
		return nil, err
	}
	return &Client{
		HttpClient: *client,
		NetParam:   opts.NetParam,
		errMap:     vaporErrMap,
	}, nil
}

// GetBlock
func (c *Client) GetBlock(id interface{}) (protocol.WrapBlock, *bc.TransactionStatus, error) {
	switch key := id.(type) {
	case string:
		return c.GetBlockByHash(key)
	case uint64:
		return c.GetBlockByHeight(key)
	default:
		return nil, nil, errors.New("unknown block id type")
	}
}

// GetBlockByHash return block by specified block hash
func (c *Client) GetBlockByHash(hash string) (protocol.WrapBlock, *bc.TransactionStatus, error) {
	return c.getRawBlock(&getRawBlockReq{BlockHash: hash})
}

// GetBlockByHeight return block by specified block height
func (c *Client) GetBlockByHeight(height uint64) (protocol.WrapBlock, *bc.TransactionStatus, error) {
	return c.getRawBlock(&getRawBlockReq{BlockHeight: height})
}

type getBlockCountResp struct {
	BlockCount uint64 `json:"block_count"`
}

// GetBlockCount return the best block height of connected node
func (c *Client) GetBlockCount() (uint64, error) {
	url := "/get-block-count"
	res := &getBlockCountResp{}
	return res.BlockCount, c.Request(url, nil, res)
}

type submitTxReq struct {
	Tx interface{} `json:"raw_transaction"`
}

type submitTxResp struct {
	TxID string `json:"tx_id"`
}

// SubmitTx submit transaction to node
func (c *Client) SubmitTx(tx interface{}) (string, error) {
	url := "/submit-transaction"
	payload, err := json.Marshal(submitTxReq{Tx: tx})
	if err != nil {
		return "", errors.Wrap(err, "json marshal")
	}

	address, err := getFirstBytomTxInAddress(tx, c.NetParam)
	if err != nil {
		return "", err
	}

	ress := make([]*submitTxResp, 3)
	errs := make([]error, 3)
	var group sync.WaitGroup

	if len(address) == 0 {
		// Polling 3 times when there is no address
		for i := 0; i < 3; i++ {
			group.Add(1)
			go func(n int) {
				ress[n] = &submitTxResp{}
				errs[n] = c.Request(url, payload, ress[n])
				group.Done()
			}(i)
		}
	} else {
		// Mapping addresses to 3 different nodes
		codes := []int{balancer.HashCode(address + "1"), balancer.HashCode(address + "2"), balancer.HashCode(address + "3")}
		urls := make([]string, 3)
		backends := c.Balancer.Backends()

		backends.RLock()
		length := backends.Len()
		for i, code := range codes {
			n := code % length
			backend, ok := backends.Get(n)
			if !ok {
				errs[i] = errors.New("cannot find node")
			} else {
				urls[i] = backend.URL
			}
		}
		backends.RUnlock()

		for i, u := range urls {
			if len(u) == 0 {
				continue
			}

			if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
				urls[i] = balancer.URLJoin(u, url)
			} else {
				urls[i] = balancer.URLJoin("http://", u, url)
			}

			group.Add(1)
			go func(n int) {
				ress[n] = &submitTxResp{}
				errs[n] = c.Request(urls[n], payload, ress[n])
				group.Done()
			}(i)
		}
	}

	group.Wait()
	for _, err := range errs {
		if err != nil {
			return "", err
		}
	}
	return ress[0].TxID, errs[0]
}

type response struct {
	Status    string          `json:"status"`
	Code      string          `json:"code"`
	Data      json.RawMessage `json:"data"`
	ErrDetail string          `json:"error_detail"`
}

// Request send http request to node
func (c *Client) Request(url string, payload []byte, respData interface{}) error {
	resp := &response{}
	if err := c.Post(url, payload, resp); err != nil {
		return err
	}

	if resp.Status != "success" {
		if err, ok := c.errMap[resp.Code]; ok {
			return err
		}
		return errors.New(resp.ErrDetail)
	}

	return json.Unmarshal(resp.Data, respData)
}

// NetInfo indicate net information
type NetInfo struct {
	Listening         bool   `json:"listening"`
	Syncing           bool   `json:"syncing"`
	Mining            bool   `json:"mining"`
	NodeXPub          string `json:"node_xpub"`
	FedAddress        string `json:"federation_address"`
	PeerCount         int    `json:"peer_count"`
	CurrentBlock      uint64 `json:"current_block"`
	IrreversibleBlock uint64 `json:"irreversible_block"`
	HighestBlock      uint64 `json:"highest_block"`
	NetWorkID         string `json:"network_id"`
}

// GetNodeInfo get node info
func (c *Client) GetNodeInfo() (*NetInfo, error) {
	url := "/net-info"
	res := &NetInfo{}
	return res, c.Request(url, nil, res)
}

type getRawBlockReq struct {
	BlockHeight uint64 `json:"block_height"`
	BlockHash   string `json:"block_hash"`
}

type getRawBlockResp struct {
	RawBlock          *types.Block          `json:"raw_block"`
	TransactionStatus *bc.TransactionStatus `json:"transaction_status"`
}

func (c *Client) getRawBlock(req *getRawBlockReq) (protocol.WrapBlock, *bc.TransactionStatus, error) {
	url := "/get-raw-block"
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, nil, errors.Wrap(err, "json marshal")
	}

	res := &getRawBlockResp{}
	if err := c.Request(url, payload, res); err != nil {
		return nil, nil, err
	}

	// Judge whether other nodes in the node list have forwarded the hash block.
	// If not, call '/submit-block' to send the block to the node not cached and cache it.
	c.Balancer.Backends().Range(func(index int, backend *balancer.Backend) bool {
		go func() {
			blockHash := res.RawBlock.Hash()
			val, ok := backend.Cache.Get(blockHash)
			flag := val.(bool)
			if !ok || !flag {
				if result, err := c.submitBlock(&submitBlockReq{Block: res.RawBlock}); err != nil {
					//todo
				} else {
					backend.Cache.Add(blockHash, result)
				}
			}
		}()
		return true
	})

	return vapor.NewWrapBlock(res.RawBlock), res.TransactionStatus, nil
}

// submitBlockReq is req struct for submit-block API
type submitBlockReq struct {
	Block *types.Block `json:"raw_block"`
}

func (c *Client) submitBlock(req *submitBlockReq) (bool, error) {
	url := "/submit-block"
	payload, err := json.Marshal(req)
	if err != nil {
		return false, errors.Wrap(err, "json marshal")
	}

	var res bool
	if err := c.Request(url, payload, &res); err != nil {
		return false, err
	}

	return res, nil
}

func getFirstBytomTxInAddress(tx interface{}, netParam string) (string, error) {
	switch val := tx.(type) {
	case types.Tx:
		return getAddressFromTxInput(&val, 0, netParam)
	case *types.Tx:
		return getAddressFromTxInput(val, 0, netParam)
	default:
		return "", nil
	}
}

func getAddressFromTxInput(tx *types.Tx, i int, netParam string) (string, error) {
	orig := tx.Inputs[i]
	id := tx.Tx.InputIDs[i]
	e := tx.Entries[id]
	switch e.(type) {
	case *vprbc.VetoInput, *vprbc.Spend:
		controlProgram := orig.ControlProgram()
		return getAddressFromControlProgram(controlProgram, netParam, false)
	case *vprbc.CrossChainInput:
		controlProgram := orig.ControlProgram()
		return getAddressFromControlProgram(controlProgram, netParam, true)
	default:
		return "", nil
	}
}

func getAddressFromControlProgram(program []byte, netParam string, isMainchain bool) (string, error) {
	netParams := vpr.NetParams(netParam)
	if isMainchain {
		netParams = consensus.BytomMainNetParams(netParams)
	}

	return vpr.ScriptToAddress(program, netParams)
}
