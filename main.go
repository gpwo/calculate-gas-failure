package main

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
)

var localHost = false

func main() {
	wallet := ""
	from := ""
	to := ""
	if localHost {
		//get wallet address from anvil
		wallet = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
		from = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
		to = "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
	} else {
		wallet = "0x478d21c5167CB66AdEDAFA8D72D1f3757F6e6206"
		from = "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"
		to = "0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9F"
	}

	contract := "0xcBc57F275dB5fd2F4da20882fCa443B9cd302eCD"
	token := "0x46101fbe580940c7dd2d2879662bc98954b5edd1"
	amount := 17720

	gas := gasCalculated(wallet, from, to, contract, token, amount)
	fmt.Println("gas", gas)
}

func gasCalculated(wallet, from, to, contract, token string, amount int) *big.Int {
	url := ""
	if localHost {
		url = "http://127.0.0.1:8545"
	} else {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
		apiKey := os.Getenv("INFURA_API_KEY")
		if apiKey == "" {
			log.Fatalf("INFURA_API_KEY environment variable is required")
		}
		url = "https://mainnet.infura.io/v3/" + apiKey
	}

	client, err := ethclient.Dial(url)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	defer client.Close()

	gasCostProposedGasWei := 857664468
	// from https://api.etherscan.io/api?module=gastracker&action=gasoracle&apikey=YourApiKeyToken called in another routine

	fmt.Println("gas.Cost.ProposedGasWei", gasCostProposedGasWei) //-->gas.Cost.ProposedGasWei 1022582486
	fmt.Println("wallet", wallet)                                 //-->0x478d21c5167CB66AdEDAFA8D72D1f3757F6e6206
	fmt.Println("from", from)                                     //-->0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D
	fmt.Println("to", to)                                         //-->0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9F
	fmt.Println("contract", contract)                             //-->0xcBc57F275dB5fd2F4da20882fCa443B9cd302eCD
	fmt.Println("token", token)                                   //-->0x46101fbe580940c7dd2d2879662bc98954b5edd1
	fmt.Println("amount", amount)                                 //-->17720

	// Specify the sender and recipient addresses
	walletAddress := common.HexToAddress(wallet)
	fromAddress := common.HexToAddress(from)
	toAddress := common.HexToAddress(to)
	contractAddress := common.HexToAddress(contract)
	tokenAddress := common.HexToAddress(token)

	contractABI, err := abi.JSON(strings.NewReader(flashLoanReceiverABI())) // Correct ABI fetching
	if err != nil {
		fmt.Println("Failed to fetch ABI:", err)
		return big.NewInt(0)
	}

	addressType, err := abi.NewType("address", "", nil)
	if err != nil {
		log.Fatalf("Failed to create address type: %v", err)
	}

	addresses := abi.Arguments{
		{Type: addressType},
		{Type: addressType},
		{Type: addressType},
	}

	packedData, err := addresses.Pack(fromAddress, toAddress, tokenAddress)
	if err != nil {
		log.Fatalf("Failed to pack addresses: %v", err)
	}

	data, err := contractABI.Pack("onFlashLoan", walletAddress, contractAddress, big.NewInt(int64(amount)), big.NewInt(0), packedData)
	if err != nil {
		fmt.Println("ABI packing failed:", err)
		return big.NewInt(0)
	}

	ctx := context.Background()

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Fatalf("Failed to suggest gas price: %v", gasPrice)
	}
	fmt.Println("gasPrice:", gasPrice)

	callMsg := ethereum.CallMsg{
		From:     fromAddress,
		To:       &contractAddress,
		GasPrice: gasPrice,
		Value:    big.NewInt(0),
		Data:     data,
	}

	// Estimate the gas required for the transaction
	gasLimit, err := client.EstimateGas(ctx, callMsg)
	if err != nil {
		fmt.Printf("Failed to estimate gas: %v.", err) //<--Failed to estimate gas: execution reverted.gas 0
		return big.NewInt(0)
	}
	fmt.Println("gasLimit:", gasLimit)
	fmt.Println("price", price(gasPrice, gasLimit)) //-->gasLimit 0

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		fmt.Println("Failed to get header:", err)
		return big.NewInt(0)
	}

	// Perform the low-level eth_call to check the transaction
	var result hexutil.Bytes
	result, err = client.CallContract(ctx, callMsg, header.Number)
	if err != nil {
		fmt.Printf("Low-level call %v\n", err)
		return big.NewInt(0)
	}

	// Decode or inspect `result` to understand the failure
	fmt.Printf("Low-level call result: %x\n", result)

	return big.NewInt(int64(gasLimit))
}

/*
The string returned from function flashLoanReceiverABI() is the ABI of the contract deployed at
https://etherscan.io/address/0xcBc57F275dB5fd2F4da20882fCa443B9cd302eCD
*/
func flashLoanReceiverABI() string {
	return `[{"inputs":[{"internalType":"address","name":"_owner","type":"address"}],"stateMutability":"nonpayable","type":"constructor"},{"inputs":[{"internalType":"address","name":"initiator","type":"address"},{"internalType":"address","name":"baseTokenAddress","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"uint256","name":"fee","type":"uint256"},{"internalType":"bytes","name":"data","type":"bytes"}],"name":"onFlashLoan","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"owner","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"newOwner","type":"address"}],"name":"updateOwner","outputs":[],"stateMutability":"nonpayable","type":"function"}]`
}

func price(gasPrice *big.Int, gasLimit uint64) *big.Int {
	return new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gasLimit))
}
