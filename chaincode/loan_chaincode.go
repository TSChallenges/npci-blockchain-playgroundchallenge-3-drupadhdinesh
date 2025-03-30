package main

import (
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type LoanContract struct {
	contractapi.Contract
}

type Loan struct {
	LoanID        string   `json:"loanID"`
	ApplicantName string   `json:"applicantName"`
	LoanAmount    float64  `json:"loanAmount"`
	TermMonths    int      `json:"termMonths"`
	InterestRate  float64  `json:"interestRate"`
	Outstanding   float64  `json:"outstanding"`
	Status        string   `json:"status"`
	Repayments    []float64 `json:"repayments"`
}

func (c *LoanContract) ApplyForLoan(ctx contractapi.TransactionContextInterface, 
	loanID, applicantName string, 
	loanAmount float64, 
	termMonths int, 
	interestRate float64) error {
	
	// Validate inputs
	if loanID == "" {
		return fmt.Errorf("loan ID cannot be empty")
	}
	if applicantName == "" {
		return fmt.Errorf("applicant name cannot be empty")
	}
	if loanAmount <= 0 {
		return fmt.Errorf("loan amount must be positive")
	}
	if termMonths <= 0 {
		return fmt.Errorf("loan term must be positive")
	}
	if interestRate < 0 {
		return fmt.Errorf("interest rate cannot be negative")
	}

	// Check if loan already exists
	existing, err := ctx.GetStub().GetState(loanID)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if existing != nil {
		return fmt.Errorf("loan ID %s already exists", loanID)
	}

	// Create new loan
	loan := Loan{
		LoanID:        loanID,
		ApplicantName: applicantName,
		LoanAmount:    loanAmount,
		TermMonths:    termMonths,
		InterestRate:  interestRate,
		Outstanding:   loanAmount,
		Status:        "APPLIED",
		Repayments:    []float64{},
	}

	// Save to ledger
	loanJSON, err := json.Marshal(loan)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(loanID, loanJSON)
}

func (c *LoanContract) ApproveLoan(ctx contractapi.TransactionContextInterface, loanID string, status string) error {
	// Validate status
	if status != "APPROVED" && status != "REJECTED" {
		return fmt.Errorf("invalid status, must be APPROVED or REJECTED")
	}

	// Get loan from ledger
	loanJSON, err := ctx.GetStub().GetState(loanID)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if loanJSON == nil {
		return fmt.Errorf("loan %s does not exist", loanID)
	}

	// Unmarshal loan
	var loan Loan
	err = json.Unmarshal(loanJSON, &loan)
	if err != nil {
		return err
	}

	// Validate current status
	if loan.Status != "APPLIED" {
		return fmt.Errorf("loan must be in APPLIED status to be approved/rejected")
	}

	// Update status
	loan.Status = status

	// Save updated loan
	updatedLoanJSON, err := json.Marshal(loan)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(loanID, updatedLoanJSON)
}

func (c *LoanContract) MakeRepayment(ctx contractapi.TransactionContextInterface, loanID string, repaymentAmount float64) error {
	// Validate repayment amount
	if repaymentAmount <= 0 {
		return fmt.Errorf("repayment amount must be positive")
	}

	// Get loan from ledger
	loanJSON, err := ctx.GetStub().GetState(loanID)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if loanJSON == nil {
		return fmt.Errorf("loan %s does not exist", loanID)
	}

	// Unmarshal loan
	var loan Loan
	err = json.Unmarshal(loanJSON, &loan)
	if err != nil {
		return err
	}

	// Validate loan status
	if loan.Status != "APPROVED" {
		return fmt.Errorf("only APPROVED loans can accept repayments")
	}

	// Validate repayment doesn't exceed outstanding
	if repaymentAmount > loan.Outstanding {
		return fmt.Errorf("repayment amount exceeds outstanding balance")
	}

	// Update loan
	loan.Outstanding -= repaymentAmount
	loan.Repayments = append(loan.Repayments, repaymentAmount)

	// Update status if fully paid
	if loan.Outstanding <= 0 {
		loan.Status = "PAID"
	}

	// Save updated loan
	updatedLoanJSON, err := json.Marshal(loan)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(loanID, updatedLoanJSON)
}

func (c *LoanContract) CheckLoanBalance(ctx contractapi.TransactionContextInterface, loanID string) (*Loan, error) {
	// Get loan from ledger
	loanJSON, err := ctx.GetStub().GetState(loanID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if loanJSON == nil {
		return nil, fmt.Errorf("loan %s does not exist", loanID)
	}

	// Unmarshal loan
	var loan Loan
	err = json.Unmarshal(loanJSON, &loan)
	if err != nil {
		return nil, err
	}

	return &loan, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(new(LoanContract))
	if err != nil {
		fmt.Printf("Error creating loan chaincode: %s", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting loan chaincode: %s", err)
	}
}
