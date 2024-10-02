package main

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/coinbase/coinbase-sdk-go/pkg/coinbase"
	"github.com/decred/base58"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

/*
This example demonstrates how to stake SOL using the CDP SDK.

To run this example, you need to:
1. Have a Solana private key and an API Key stored in a file.
2. Export the following environment variables:
	- CDP_API_KEY_PATH: the path to the file containing the API Key.
	- SOLANA_PRIVATE_KEY_PATH: the path to the file containing the Solana private key in base58 encoding.
	- SOLANA_ADDRESS: the Solana address to stake the funds.
*/

func main() {
	ctx := context.Background()

	client, err := coinbase.NewClient(
		coinbase.WithAPIKeyFromJSON(os.Getenv("CDP_API_KEY_PATH")),
	)
	if err != nil {
		log.Fatalf("error creating coinbase client: %v", err)
	}

	address := coinbase.NewExternalAddress(coinbase.SolanaDevnet, os.Getenv("SOLANA_ADDRESS"))

	// Can source devnet funds from the faucet: https://faucet.solana.com/
	stakeableBalance, err := client.GetStakeableBalance(ctx, coinbase.Sol, address)
	if err != nil {
		log.Fatalf("error getting stakeable balance: %v", err)
	}

	fmt.Printf("stakeable balance: %v\n", stakeableBalance.Amount())

	stakingOperation, err := client.BuildStakeOperation(
		ctx,
		big.NewFloat(0.1),
		coinbase.Sol,
		address,
	)
	if err != nil {
		log.Fatalf("error building stake operation: %v", err)
	}

	transaction := stakingOperation.Transactions()[0]

	fmt.Printf("staking operation: %v\n", stakingOperation.ID())
	fmt.Printf("unsigned staking transaction: %v\n", transaction.UnsignedPayload())
	privateKey, err := readPrivateKey(os.Getenv("SOLANA_PRIVATE_KEY_PATH"))

	err = stakingOperation.Sign(privateKey)
	if err != nil {
		log.Fatalf("error signing transaction: %v", err)
	}

	signedTx := transaction.SignedPayload()

	fmt.Printf("signed transaction: %s\n", signedTx)

	rawTx := transaction.Raw()
	solanaTx, ok := rawTx.(*solana.Transaction)
	if !ok {
		log.Fatal("failed to cast raw transaction to solana.Transaction")
	}

	rpcClient := rpc.New("https://api.devnet.solana.com")
	maxRetries := uint(5)
	opts := rpc.TransactionOpts{
		SkipPreflight:       false,
		MaxRetries:          &maxRetries,
		PreflightCommitment: rpc.CommitmentProcessed,
	}

	signature, err := rpcClient.SendTransactionWithOpts(ctx, solanaTx, opts)
	if err != nil {
		log.Fatalf("failed to send transaction: %v", err)
	}

	fmt.Printf("broadcasted transaction hash: %s\n", signature.String())
	fmt.Printf("transaction link: https://explorer.solana.com/tx/%s?cluster=devnet", signature.String())
}

func readPrivateKey(filePath string) (*ed25519.PrivateKey, error) {
	// Read the private key file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %v", err)
	}

	// Decode the base58 encoded private key
	privateKeyBytes := base58.Decode(string(data))
	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		log.Fatalf("invalid private key length: expected %d bytes, got %d bytes", ed25519.PrivateKeySize, len(privateKeyBytes))
	}

	// Convert the byte slice to an ed25519 private key
	privateKey := ed25519.PrivateKey(privateKeyBytes)

	return &privateKey, nil
}
