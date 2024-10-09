package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/pol"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonrollupmanager"
	polygonzkevmelderberry "github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevm"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	SequencerAdminAddress = "0xff6250d0E86A2465B0C1bF8e36409503d6a26963"
	NewSequencerAddress = "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"
	NewSequencerAddressPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	OldSequencerAddress = "0x761d53b47334bEe6612c0Bd1467FB881435375B2"
	AggregatorAdminAddress = "0xff6250d0E86A2465B0C1bF8e36409503d6a26963"
	NewAggregatorAddress = "0x70997970c51812dc3a010c7d01b50e0d17dc79c8"
	ZKEVMAddress = "0xA13Ddb14437A8F34897131367ad3ca78416d6bCa"
	RollupManagerAddr = "0x32d33D5137a7cFFb54c5Bf8371172bcEc5f310ff"
	PolAddress = "0x6a7c3F4B0651d6DA389AD1d11D962ea458cDCA70"
	URL = "http://localhost:8545"
	PolAmount = "10000000000000000000000"
	// DefaultInterval is a time interval
	DefaultWaitInterval = 2 * time.Millisecond
)
var (
	TRUSTED_AGGREGATOR_ROLE = crypto.Keccak256Hash([]byte("TRUSTED_AGGREGATOR_ROLE"))
	TRUSTED_AGGREGATOR_ROLE_ADMIN = crypto.Keccak256Hash([]byte("TRUSTED_AGGREGATOR_ROLE_ADMIN"))
	// ErrTimeoutReached is thrown when the timeout is reached and
	// because the condition is not matched
	ErrTimeoutReached = errors.New("timeout has been reached")
)
func main() {
	ctx := context.Background()
	// Connect to ethereum node
	ethClient, err := ethclient.Dial(URL)
	if err != nil {
		log.Fatalf("error connecting to %s: %+v", URL, err)
	}
	zkevm, err := polygonzkevmelderberry.NewPolygonzkevm(common.HexToAddress(ZKEVMAddress), ethClient)
	if err != nil {
		log.Fatalf("error creating Polygonzkevm client (%s). Error: %v", ZKEVMAddress, err)
	}
	rollupManager, err := polygonrollupmanager.NewPolygonrollupmanager(common.HexToAddress(RollupManagerAddr), ethClient)
	if err != nil {
		log.Fatalf("error creating NewPolygonrollupmanager client (%s). Error: %v", RollupManagerAddr, err)
	}
	err = changeSequencerAddress(ctx, ethClient, common.HexToAddress(NewSequencerAddress), common.HexToAddress(SequencerAdminAddress), zkevm)
	if err != nil {
		log.Fatal("error changing sequencer address. Error: ", err)
	}
	err = changeAggregatorAddress(ctx, ethClient, common.HexToAddress(NewAggregatorAddress), common.HexToAddress(AggregatorAdminAddress), rollupManager)
	if err != nil {
		log.Fatal("error changing sequencer address. Error: ", err)
	}
	err = sendPolTokensToNewSequencerAddress(ctx, ethClient, common.HexToAddress(NewSequencerAddress), common.HexToAddress(OldSequencerAddress))
	if err != nil {
		log.Fatal("error sending pol tokens to the new address. Error: ", err)
	}
}

type Tx struct {
	From string `json:"from"`
	To   string `json:"to"`
	Data string `json:"data"`
}
func changeSequencerAddress(ctx context.Context, ethClient *ethclient.Client, newSeqAddress, seqAdminAddress common.Address, zkevm *polygonzkevmelderberry.Polygonzkevm) error {
	err := impersonateAccount(seqAdminAddress)
	if err != nil {
		return err
	}
	err = setNewSequencerAddress(ctx, ethClient, newSeqAddress, seqAdminAddress)
	if err != nil {
		return err
	}
	err = stopImpersonatingAccount(seqAdminAddress)
	if err != nil {
		return err
	}
	// Check if the trusted sequencer address has been modified successfully 
	address, err := zkevm.TrustedSequencer(&bind.CallOpts{Pending: false})
	if err != nil {
		return err
	}
	if address != newSeqAddress {
		return fmt.Errorf("error setting new sequencer address. Expected address: %s, received address: %s", newSeqAddress.String(), address.String())
	}
	return nil
}

func changeAggregatorAddress(ctx context.Context, ethClient *ethclient.Client, newAggAddress, aggAdminAddress common.Address, rollupManager *polygonrollupmanager.Polygonrollupmanager) error {
	err := impersonateAccount(aggAdminAddress)
	if err != nil {
		return err
	}
	// Add new Address to TRUSTED_AGGREGATOR_ROLE
	err = setNewAggregatorAddress(ctx, ethClient, newAggAddress, aggAdminAddress)
	if err != nil {
		return err
	}
	err = stopImpersonatingAccount(aggAdminAddress)
	if err != nil {
		return err
	}
	// Check if trusted aggregator address has been modified successfully
	ok, err := rollupManager.HasRole(&bind.CallOpts{Pending: false}, TRUSTED_AGGREGATOR_ROLE, newAggAddress)
	if err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("error setting new trusted aggregator address. New aggregator address (%s) hasn't the role TRUSTED_AGGREGATOR_ROLE (%s)", newAggAddress.String(), TRUSTED_AGGREGATOR_ROLE.String())
	}
	return nil
}

func sendPolTokensToNewSequencerAddress(ctx context.Context, ethClient *ethclient.Client, newSeqAddress, oldSequencerAddress common.Address) error {
	err := impersonateAccount(oldSequencerAddress)
	if err != nil {
		return err
	}	
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(NewSequencerAddressPrivateKey, "0x"))
	if err != nil {
		return err
	}
	chainID, err := ethClient.ChainID(ctx)
	if err != nil {
		return err
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Error("Error getting signer. Error: ", err)
		return err
	}
	amount, _ := big.NewInt(0).SetString(PolAmount, 0)
	err = sendPolTokens(ctx, ethClient, newSeqAddress, oldSequencerAddress, amount)
	if err != nil {
		return err
	}
	p, err := pol.NewPol(common.HexToAddress(PolAddress), ethClient)
	if err != nil {
		log.Errorf("error creating NewPol client (%s). Error: %v", PolAddress, err)
		return err
	}
	tx, err := p.Approve(auth, common.HexToAddress(ZKEVMAddress), amount)
	if err != nil {
		return err
	}
	log.Debug("Approve pol tx sent. TxHash: ", tx.Hash())
	err = stopImpersonatingAccount(oldSequencerAddress)
	if err != nil {
		return err
	}
	return nil
}

func setNewAggregatorAddress(ctx context.Context, ethClient *ethclient.Client, newAggAddress, aggAdminAddress common.Address) error {
	a, _ := polygonrollupmanager.PolygonrollupmanagerMetaData.GetAbi()
	input, err := a.Pack("grantRole", TRUSTED_AGGREGATOR_ROLE, newAggAddress)
	if err != nil {
		log.Error("error packing call grantRole for trusted aggregator address. Error: ", err)
		return err
	}
	tx := Tx {
		From: aggAdminAddress.String(),
		To:   RollupManagerAddr,
		Data: fmt.Sprintf("0x%s",common.Bytes2Hex(input)),
	}
	body := RequestBody {
		Jsonrpc: "2.0",
		Method:  "eth_sendTransaction",
		Params:  []interface{}{tx},
		Id:      1,
	}
	reqBody, err := json.Marshal(body)
    if err != nil {
		log.Errorf("error marshalling in setting new trusted aggregator. Error: %v", err)
        return err
    }
	bodyBytes, err := callRpc(reqBody)
	if err != nil {
		log.Errorf("error calling RPC in setting new trusted aggregator. Error: %v", err)
		return err
	}
	var respBody ResponseBody
	err = json.Unmarshal(bodyBytes, &respBody)
	if err != nil {
		log.Errorf("error unmarshalling response body in stop setting new trusted aggregator. Error: %v", err)
		return err
	}
	if respBody.Error != nil {
		return fmt.Errorf("error stop setting new trusted aggregator. Error: %s", respBody.Error.Message)
	}
	log.Debugf("NewAggregator transaction response: %+v", *respBody.Result)
	// Wait until tx is mined
	timeout := 20 * time.Second
	_, err = WaitTxReceipt(ctx, common.HexToHash((*respBody.Result).(string)), timeout, ethClient)
	return err
}

func setNewSequencerAddress(ctx context.Context, ethClient *ethclient.Client, newSeqAddress, seqAdminAddress common.Address) error {
	a, _ := polygonzkevmelderberry.PolygonzkevmMetaData.GetAbi()
	input, err := a.Pack("setTrustedSequencer", newSeqAddress)
	if err != nil {
		log.Error("error packing call setTrustedSequencer. Error: ", err)
		return err
	}
	tx := Tx {
		From: seqAdminAddress.String(),
		To:   ZKEVMAddress,
		Data: fmt.Sprintf("0x%s",common.Bytes2Hex(input)),
	}
	body := RequestBody {
		Jsonrpc: "2.0",
		Method:  "eth_sendTransaction",
		Params:  []interface{}{tx},
		Id:      1,
	}
	reqBody, err := json.Marshal(body)
    if err != nil {
		log.Errorf("error marshalling in setting new trusted sequencer. Error: %v", err)
        return err
    }
	bodyBytes, err := callRpc(reqBody)
	if err != nil {
		log.Errorf("error calling RPC in setting new trusted sequencer. Error: %v", err)
		return err
	}
	var respBody ResponseBody
	err = json.Unmarshal(bodyBytes, &respBody)
	if err != nil {
		log.Errorf("error unmarshalling response body in stop setting new trusted sequencer. Error: %v", err)
		return err
	}
	if respBody.Error != nil {
		return fmt.Errorf("error stop setting new trusted sequencer. Error: %s", respBody.Error.Message)
	}
	log.Debugf("NewSequencer transaction response: %+v", *respBody.Result)
	// Wait until tx is mined
	timeout := 20 * time.Second
	_, err = WaitTxReceipt(ctx, common.HexToHash((*respBody.Result).(string)), timeout, ethClient)
	return err
}
func sendPolTokens(ctx context.Context, ethClient *ethclient.Client, newSeqAddress, oldSequencerAddress common.Address, amount *big.Int) error {
	a, _ := pol.PolMetaData.GetAbi()
	input, err := a.Pack("transfer", newSeqAddress, amount)
	if err != nil {
		log.Error("error packing call transfer. Error: ", err)
		return err
	}
	tx := Tx {
		From: oldSequencerAddress.String(),
		To:   PolAddress,
		Data: fmt.Sprintf("0x%s",common.Bytes2Hex(input)),
	}
	body := RequestBody {
		Jsonrpc: "2.0",
		Method:  "eth_sendTransaction",
		Params:  []interface{}{tx},
		Id:      1,
	}
	reqBody, err := json.Marshal(body)
    if err != nil {
		log.Errorf("error marshalling in send pol tokens. Error: %v", err)
        return err
    }
	bodyBytes, err := callRpc(reqBody)
	if err != nil {
		log.Errorf("error calling RPC in send pol tokens. Error: %v", err)
		return err
	}
	var respBody ResponseBody
	err = json.Unmarshal(bodyBytes, &respBody)
	if err != nil {
		log.Errorf("error unmarshalling response body in stop send pol tokens. Error: %v", err)
		return err
	}
	if respBody.Error != nil {
		return fmt.Errorf("error stop send pol tokens. Error: %s", respBody.Error.Message)
	}
	log.Debugf("Send Pol transaction response: %+v", *respBody.Result)
	// Wait until tx is mined
	timeout := 20 * time.Second
	_, err = WaitTxReceipt(ctx, common.HexToHash((*respBody.Result).(string)), timeout, ethClient)
	return err
}

type RequestBody struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}
type ResponseBody struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  *interface{}  `json:"result"`
	Id      int    `json:"id"`
	Error *struct {
		Code int       `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
func impersonateAccount(addr common.Address) error {
	body := RequestBody {
		Jsonrpc: "2.0",
		Method:  "hardhat_impersonateAccount",
		Params:  []interface{}{addr.String()},
		Id:      1,
	}
	reqBody, err := json.Marshal(body)
    if err != nil {
		log.Errorf("error marshalling in Impersonating Account. Error: %v", err)
        return err
    }
	bodyBytes, err := callRpc(reqBody)
	if err != nil {
		log.Errorf("error calling RPC in Impersonating Account. Error: %v", err)
		return err
	}
	var respBody ResponseBody
	err = json.Unmarshal(bodyBytes, &respBody)
	if err != nil {
		log.Errorf("error unmarshalling response body in Impersonating Account. Error: %v", err)
		return err
	}
	if respBody.Error != nil {
		return fmt.Errorf("error impersonating account. Error: %s", respBody.Error.Message)
	}
	return nil
}

func stopImpersonatingAccount(addr common.Address) error {
	body := RequestBody {
		Jsonrpc: "2.0",
		Method:  "hardhat_stopImpersonatingAccount",
		Params:  []interface{}{addr.String()},
		Id:      1,
	}
	reqBody, err := json.Marshal(body)
    if err != nil {
		log.Errorf("error marshalling in stop impersonating account. Error: %v", err)
        return err
    }
	bodyBytes, err := callRpc(reqBody)
	if err != nil {
		log.Errorf("error calling RPC in stop impersonating account. Error: %v", err)
		return err
	}
	var respBody ResponseBody
	err = json.Unmarshal(bodyBytes, &respBody)
	if err != nil {
		log.Errorf("error unmarshalling response body in stop impersonating account. Error: %v", err)
		return err
	}
	if respBody.Error != nil {
		return fmt.Errorf("error stop impersonating account. Error: %s", respBody.Error.Message)
	}
	return nil
}

func callRpc(reqBody []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, URL, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Errorf("error creating newRequest. Error: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("error doing the request. Error: %v", err)
		return nil, err
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Error("error closing response body")
		}
	}()

	var bodyBytes []byte
	if res.StatusCode == http.StatusOK {
		bodyBytes, err = io.ReadAll(res.Body)
		if err != nil {
			log.Errorf("error reading response body. Error: %v", err)
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("error in http request. Status code received: %d", res.StatusCode)
	}
	return bodyBytes, nil
}

func WaitTxReceipt(ctx context.Context, txHash common.Hash, timeout time.Duration, client *ethclient.Client) (*types.Receipt, error) {
	if client == nil {
		return nil, fmt.Errorf("client is nil")
	}
	var receipt *types.Receipt
	pollErr := Poll(DefaultWaitInterval, timeout, func() (bool, error) {
		var err error
		receipt, err = client.TransactionReceipt(ctx, txHash)
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				time.Sleep(time.Second)
				return false, nil
			} else {
				return false, err
			}
		}
		if receipt != nil {
			return true, nil
		} else {
			return false, nil
		}
	})
	if pollErr != nil {
		return nil, pollErr
	}
	return receipt, nil
}

// ConditionFunc is a generic function
type ConditionFunc func() (done bool, err error)

// Poll retries the given condition with the given interval until it succeeds
// or the given deadline expires.
func Poll(interval, deadline time.Duration, condition ConditionFunc) error {
	timeout := time.After(deadline)
	tick := time.NewTicker(interval)

	for {
		select {
		case <-timeout:
			return ErrTimeoutReached
		case <-tick.C:
			ok, err := condition()
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		}
	}
}