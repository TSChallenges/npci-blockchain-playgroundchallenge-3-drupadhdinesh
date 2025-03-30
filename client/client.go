package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

type Loan struct {
	LoanID        string    `json:"loanID"`
	ApplicantName string    `json:"applicantName"`
	LoanAmount    float64   `json:"loanAmount"`
	TermMonths    int       `json:"termMonths"`
	InterestRate  float64   `json:"interestRate"`
	Outstanding   float64   `json:"outstanding"`
	Status        string    `json:"status"`
	Repayments    []float64 `json:"repayments"`
}

func populateWallet(wallet *gateway.Wallet) error {
	credPath := filepath.Join(
		"..",
		"..",
		"test-network",
		"organizations",
		"peerOrganizations",
		"org1.example.com",
		"users",
		"User1@org1.example.com",
		"msp",
	)

	certPath := filepath.Join(credPath, "signcerts", "cert.pem")
	// read the certificate pem
	cert, err := os.ReadFile(filepath.Clean(certPath))
	if err != nil {
		return err
	}

	keyDir := filepath.Join(credPath, "keystore")
	files, err := os.ReadDir(keyDir)
	if err != nil {
		return err
	}
	if len(files) != 1 {
		return fmt.Errorf("keystore folder should have contain one file")
	}
	keyPath := filepath.Join(keyDir, files[0].Name())
	key, err := os.ReadFile(filepath.Clean(keyPath))
	if err != nil {
		return err
	}

	identity := gateway.NewX509Identity("Org1MSP", string(cert), string(key))
	return wallet.Put("appUser", identity)
}

func main() {
	log.Println("============ Starting Bank Loan Client ============")

	err := os.Setenv("DISCOVERY_AS_LOCALHOST", "true")
	if err != nil {
		log.Fatalf("Error setting DISCOVERY_AS_LOCALHOST: %v", err)
	}

	walletPath := "wallet"
	wallet, err := gateway.NewFileSystemWallet(walletPath)
	if err != nil {
		log.Fatalf("Failed to create wallet: %v", err)
	}

	if !wallet.Exists("appUser") {
		err = populateWallet(wallet)
		if err != nil {
			log.Fatalf("Failed to populate wallet: %v", err)
		}
	}

	ccpPath := filepath.Join(
		"..",
		"..",
		"test-network",
		"organizations",
		"peerOrganizations",
		"org1.example.com",
		"connection-org1.yaml",
	)

	gw, err := gateway.Connect(
		gateway.WithConfig(config.FromFile(filepath.Clean(ccpPath))),
		gateway.WithIdentity(wallet, "appUser"),
	)
	if err != nil {
		log.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer gw.Close()

	network, err := gw.GetNetwork("mychannel")
	if err != nil {
		log.Fatalf("Failed to get network: %v", err)
	}

	contract := network.GetContract("loancontract")

	// Test Case 1: Apply for a loan
	log.Println("--> Test Case 1: Apply for a loan")
	_, err = contract.SubmitTransaction(
		"ApplyForLoan", 
		"loan1", 
		"John Doe", 
		"5000", 
		"12", 
		"5.5",
	)
	if err != nil {
		log.Fatalf("Failed to apply for loan: %v", err)
	}
	log.Println("Loan successfully applied")

	// Test Case 2: Check loan status
	log.Println("--> Test Case 2: Check loan status")
	result, err := contract.EvaluateTransaction("CheckLoanBalance", "loan1")
	if err != nil {
		log.Fatalf("Failed to evaluate transaction: %v", err)
	}
	var loan Loan
	err = json.Unmarshal(result, &loan)
	if err != nil {
		log.Fatalf("Failed to parse loan data: %v", err)
	}
	log.Printf("Loan Status: %s\n", loan.Status)
	log.Printf("Outstanding Balance: %.2f\n", loan.Outstanding)

	log.Println("============ Client completed successfully ============")
}