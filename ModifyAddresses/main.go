package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	polygonzkevmelderberry "github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevm"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonrollupmanager"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	SequencerAdminAddress = "0xff6250d0E86A2465B0C1bF8e36409503d6a26963"
	NewSequencerAddress = "0x2536C2745Ac4A584656A830f7bdCd329c94e8F30"
	AggregatorAdminAddress = "0xff6250d0E86A2465B0C1bF8e36409503d6a26963"
	NewAggregatorAddress = "0xff6250d0E86A2465B0C1bF8e36409503d6a26963"
	ZKEVMAddress = "0xA13Ddb14437A8F34897131367ad3ca78416d6bCa"
	RollupManagerAddr = "0x32d33D5137a7cFFb54c5Bf8371172bcEc5f310ff"
	URL = "http://localhost:8545"
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
	// auth, err := generateRandomAuth(ctx, ethClient)
	// if err != nil {
	// 	log.Fatalf("error generating random auth. Error: %v", err)
	// }
	err = changeSequencerAddress(ctx, ethClient, common.HexToAddress(NewSequencerAddress), common.HexToAddress(SequencerAdminAddress), zkevm)
	if err != nil {
		log.Fatal("error changing sequencer address. Error: ", err)
	}
	err = changeAggregatorAddress(ctx, ethClient, common.HexToAddress(NewAggregatorAddress), common.HexToAddress(AggregatorAdminAddress), rollupManager)
	if err != nil {
		log.Fatal("error changing sequencer address. Error: ", err)
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

// func generateRandomAuth(ctx context.Context, ethClient *ethclient.Client) (bind.TransactOpts, error) {
// 	privateKey, err := crypto.GenerateKey()
// 	if err != nil {
// 		return bind.TransactOpts{}, errors.New("failed to generate a private key to estimate L1 txs")
// 	}
// 	chainID, err := ethClient.ChainID(ctx)
// 	if err != nil {
// 		return bind.TransactOpts{}, errors.New("failed to read chainID to estimate L1 txs")
// 	}
// 	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
// 	if err != nil {
// 		return bind.TransactOpts{}, errors.New("failed to generate a fake authorization to estimate L1 txs")
// 	}
// 	return *auth, nil
// }

// // WaitTxToBeMined waits until a tx has been mined or the given timeout expires.
// func WaitTxToBeMined(parentCtx context.Context, ethClient *ethclient.Client, tx *types.Transaction, timeout time.Duration) error {
// 	ctx, cancel := context.WithTimeout(parentCtx, timeout)
// 	defer cancel()
// 	receipt, err := bind.WaitMined(ctx, ethClient, tx)
// 	if errors.Is(err, context.DeadlineExceeded) {
// 		return err
// 	} else if err != nil {
// 		log.Errorf("error waiting tx %s to be mined: %w", tx.Hash(), err)
// 		return err
// 	}
// 	if receipt.Status == types.ReceiptStatusFailed {
// 		// Get revert reason
// 		reason, reasonErr := RevertReason(ctx, ethClient, tx, receipt.BlockNumber)
// 		if reasonErr != nil {
// 			reason = reasonErr.Error()
// 		}
// 		return fmt.Errorf("transaction has failed, reason: %s, receipt: %+v. tx: %+v, gas: %v", reason, receipt, tx, tx.Gas())
// 	}
// 	log.Debug("Transaction successfully mined: ", tx.Hash())
// 	return nil
// }

// // RevertReason returns the revert reason for a tx that has a receipt with failed status
// func RevertReason(ctx context.Context, ethClient *ethclient.Client, tx *types.Transaction, blockNumber *big.Int) (string, error) {
// 	if tx == nil {
// 		return "", nil
// 	}

// 	from, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
// 	if err != nil {
// 		signer := types.LatestSignerForChainID(tx.ChainId())
// 		from, err = types.Sender(signer, tx)
// 		if err != nil {
// 			return "", err
// 		}
// 	}
// 	msg := ethereum.CallMsg{
// 		From: from,
// 		To:   tx.To(),
// 		Gas:  tx.Gas(),

// 		Value: tx.Value(),
// 		Data:  tx.Data(),
// 	}
// 	hex, err := ethClient.CallContract(ctx, msg, blockNumber)
// 	if err != nil {
// 		return "", err
// 	}

// 	unpackedMsg, err := abi.UnpackRevert(hex)
// 	if err != nil {
// 		log.Warnf("failed to get the revert message for tx %v: %v", tx.Hash(), err)
// 		return "", errors.New("execution reverted")
// 	}

// 	return unpackedMsg, nil
// }

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