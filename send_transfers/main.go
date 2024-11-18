package main

import (
	"context"
	"fmt"
	"math/big"

	"github.com/0xPolygonHermez/zkevm-node/hex"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	var networks = []struct {
		Name       string
		URL        string
		ChainID    uint64
		PrivateKey string
	}{
		//{Name: "Local L1", URL: operations.DefaultL1NetworkURL, ChainID: operations.DefaultL1ChainID, PrivateKey: operations.DefaultSequencerPrivateKey},
		{Name: "Local L2", URL: "http://localhost:8123", ChainID: 2442, PrivateKey: "0x<privKey>"},
	}

	for _, network := range networks {
		ctx := context.Background()

		log.Infof("connecting to %v: %v", network.Name, network.URL)
		client, err := ethclient.Dial(network.URL)
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("connected")

		auth := operations.MustGetAuth(network.PrivateKey, network.ChainID)
		if err != nil {
			log.Fatal(err)
		}

		const receiverAddr = "0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D"

		balance, err := client.BalanceAt(ctx, auth.From, nil)
		if err != nil {
			log.Fatal(err)
		}
		log.Debugf("ETH Balance for %v: %v", auth.From, balance)

		transferAmount := big.NewInt(1)
		log.Debugf("Transfer Amount: %v", transferAmount)

		nonce, err := client.NonceAt(ctx, auth.From, nil)
		if err != nil {
			log.Fatal(err)
		}
		// var lastTxHash common.Hash
		for i := 0; i < 10; i++ {
			nonce := nonce + uint64(i)
			log.Debugf("Sending TX to transfer ETH")
			to := common.HexToAddress(receiverAddr)
			tx, err := ethTransfer(ctx, client, auth, to, transferAmount, &nonce)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("tx sent: ", tx.Hash().String())
			// lastTxHash = tx.Hash()
		}

		// err = operations.WaitTxToBeMined(client, lastTxHash, txTimeout)
		// chkErr(err)
	}
}

func ethTransfer(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts, to common.Address, amount *big.Int, nonce *uint64) (*types.Transaction, error) {
	if nonce == nil {
		log.Infof("reading nonce for account: %v", auth.From.Hex())
		var err error
		n, err := client.NonceAt(ctx, auth.From, nil)
		log.Infof("nonce: %v", n)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		nonce = &n
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Error(err)
		return nil, err
	}
	gasLimit, err := client.EstimateGas(context.Background(), ethereum.CallMsg{To: &to})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	tx := types.NewTransaction(*nonce, to, amount, gasLimit, gasPrice, nil)

	signedTx, err := auth.Signer(auth.From, tx)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.Infof("sending transfer tx")
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.Infof("tx sent: %v", signedTx.Hash().Hex())

	rlp, err := signedTx.MarshalBinary()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.Infof("tx rlp: %v", hex.EncodeToHex(rlp))

	return signedTx, nil
}

