package contract

import (
	"encoding/json"
	"log"
	"os"
	"sort"
	"time"

	"github.com/pkg/errors"
)

type DeployedContracts map[string]*DeployedContract

type DeployedContract struct {
	CompiledContract
	Name          string    `json:"name"`
	DeployName    string    `json:"deployName"`
	Address       Bytes     `json:"address"`
	TransactionID Bytes     `json:"txid"`
	CreatedAt     time.Time `json:"createdAt"`
	Confirmed     bool      `json:"confirmed"`
}

type ContractsRepository struct {
	filepath  string
	contracts DeployedContracts
}

func OpenContractsRepository(filepath string) (repo *ContractsRepository, err error) {
	f, err := os.Open(filepath)
	if os.IsNotExist(err) {
		return &ContractsRepository{
			filepath:  filepath,
			contracts: make(DeployedContracts),
		}, nil
	}

	if err != nil {
		return
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	contracts := make(DeployedContracts)
	err = dec.Decode(&contracts)
	if err != nil {
		return
	}

	return &ContractsRepository{
		filepath:  filepath,
		contracts: contracts,
	}, nil
}

func (r *ContractsRepository) Get(name string) (*DeployedContract, bool) {
	contract, ok := r.contracts[name]
	return contract, ok
}

func (r *ContractsRepository) Len() int {
	return len(r.contracts)
}

func (r *ContractsRepository) ConfirmContract(contract *DeployedContract) {

}

func (r *ContractsRepository) UnconfirmedContracts() []*DeployedContract {
	var contracts []*DeployedContract

	for _, contract := range r.contracts {
		if !contract.Confirmed {
			contracts = append(contracts, contract)
		}
	}

	return contracts
}

func (r *ContractsRepository) SortedContracts() []*DeployedContract {
	var contracts []*DeployedContract

	for _, contract := range r.contracts {
		contracts = append(contracts, contract)
	}

	sort.Slice(contracts, func(i, j int) bool {
		c1 := contracts[i]
		c2 := contracts[j]

		return c1.CreatedAt.Unix() < c2.CreatedAt.Unix()
	})

	return contracts
}

func (r *ContractsRepository) Exists(name string) bool {
	_, found := r.contracts[name]
	return found
}

func (r *ContractsRepository) Confirm(name string) (err error) {
	c, found := r.contracts[name]
	if !found {
		return errors.Errorf("Cannot unconfirm unknown contract %s", name)
	}

	c.Confirmed = true

	return nil
}

func (r *ContractsRepository) Set(name string, c *DeployedContract) (err error) {
	r.contracts[name] = c
	return nil
}

func (r *ContractsRepository) Commit() (err error) {
	// TODO. do write & swap instead of truncat?
	f, err := os.Create(r.filepath)
	if err != nil {
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	return enc.Encode(r.contracts)
}

// Confirm checks the RPC server to see if all the contracts
// are confirmed by the blockchain.
func (r *ContractsRepository) ConfirmAll(updateProgress func(i, total int), confirmer func(c *DeployedContract) error) (err error) {
	contracts := r.UnconfirmedContracts()

	total := len(contracts)

	if updateProgress != nil {
		updateProgress(0, total)
	}

	for i, contract := range contracts {
		contract := contract

		err := confirmer(contract)
		if err != nil {
			log.Println("err", err)
		}

		if updateProgress != nil {
			updateProgress(i+1, total)
		}
	}

	err = r.Commit()
	if err != nil {
		return
	}

	return
}
