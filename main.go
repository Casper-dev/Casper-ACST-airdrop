package main

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"sync"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/core/types"
	"time"
	"math/rand"
	"github.com/Casper-dev/Casper-ACST-airdrop/acst"
	"fmt"
	"context"
	"bufio"
	"os"
	"math/big"
)

const TokensToSend = 50

func main() {
	if len(os.Args) > 1 {
		fmt.Println("Enter your private key without 0x")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		privKey := scanner.Text()

		addresses, err := os.Open(os.Args[1])
		if err != nil {
			fmt.Println(err)
			os.Exit( 1)
		}
		addrScanner := bufio.NewScanner(addresses)
		SendEth(addrScanner, privKey)
	} else {
		fmt.Println("addresses.txt was not supplied")
		os.Exit( 1)
	}
}

func SendEth(scanner *bufio.Scanner, privKey string) {

	for scanner.Scan() {
	sc, client, auth, err := InitSC(privKey)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
		address := scanner.Text()
		if address[0] == '0' && address[1] == 'x' {
			address = address[2:]
		}
		fmt.Println(address)
		tokens := big.NewInt(TokensToSend)
		tokens.SetString("50000000000000000000", 10)
		fmt.Println(tokens.String())
		auth.Nonce.Add(auth.Nonce, big.NewInt(1))
		tx, err := sc.Transfer(auth, common.HexToAddress(address), tokens)
		for err != nil {
			auth.Nonce.Add(auth.Nonce, big.NewInt(1))
			
			fmt.Println(err)
			time.Sleep(time.Millisecond * time.Duration(100+rand.Intn(1000)))
			tx, err =  sc.Transfer(auth, common.HexToAddress(address), tokens)
		}
		_, err = MineTX(tx, client)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(tx.Hash().String())
		time.Sleep(time.Second * 30)
	}
}

// with no 0x
const ContractAddress = "f4d9f469297d7c6a8c3962aa04ab37d6cfa66ee7"

var mu sync.Mutex


func InitSC(privKey string) (*acst.AdviserCasperToken, *ethclient.Client, *bind.TransactOpts, error) {
	client, err := ethclient.Dial("https://mainnet.infura.io/S60XoLCeh1Z6LM1XiU7G")
	if err != nil {
		return nil, nil, nil, err
	}

	contractAddress := common.HexToAddress(ContractAddress)
	casperSClient, err := acst.NewAdviserCasperToken(contractAddress, client)
	if err != nil {
		return nil, nil, nil, err
	}
	key, err := crypto.HexToECDSA(privKey)
	if err != nil {
		return nil, nil, nil, err
	}
	auth := bind.NewKeyedTransactor(key)	
	
	//fmt.Println(client.PendingBalanceAt(context.Background(), auth.From))
	//auth.GasPrice = big.NewInt(250000000)
	// TODO add comment about constant
	auth.GasLimit = uint64(1500000)
	return casperSClient, client, auth, nil
}

func ValidateMineTX(txGet func() (tx *types.Transaction, err error), client *ethclient.Client) (string, error) {
	tx, err := txGet()
	for err != nil {
		fmt.Println(err)
		time.Sleep(time.Millisecond * time.Duration(100+rand.Intn(3000)))
		tx, err = txGet()
	}
	fmt.Println("Pending TX: 0x%x\n", tx.Hash())
	return MineTX(tx, client)
}

func MineTX(tx *types.Transaction, client *ethclient.Client) (data string, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered in MineTX():", r)
		}
	}()

	fmt.Printf("Gas %d; Gas price %d", tx.Gas(), tx.GasPrice())
	c := 0
	for {
		rxt, pending, err := client.TransactionByHash(context.Background(), tx.Hash())
		if err != nil {
			c++;
			if c > 2 { 
				fmt.Println(err)
				return "", err
			}
		} else {
			fmt.Println("Waiting for TX to mine")
		}
		if !pending {
			fmt.Println("Waiting a second for the receipt")
			time.Sleep(1 * time.Second)
			fmt.Println("getting receipt")
			receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
			if err != nil {
				fmt.Println(err)
				return "", err
			}
			fmt.Println(rxt.String())
			fmt.Println("receipt: %s %s", receipt.Status, receipt.String())
			if len(receipt.Logs) > 0 {
				for _, receiptLog := range receipt.Logs {
					data += string(receiptLog.Data)
				}
				fmt.Println("data: %s", data)
			}
			break
		}
		time.Sleep(2500 * time.Millisecond)

	}
	fmt.Println("Tx succesfully mined")
	return data, nil
}
