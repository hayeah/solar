package qtum

import (
	"net/url"
	"time"

	"github.com/hayeah/solar/contract"
	"github.com/pkg/errors"
	"math/rand"
)

type Deployer struct {
	rpc *rpc
	*contract.ContractsRepository
}

func NewDeployer(rpcURL *url.URL, repo *contract.ContractsRepository) (*Deployer, error) {
	return &Deployer{
		rpc: &rpc{
			BaseURL: rpcURL,
		},
		ContractsRepository: repo,
	}, nil
}

func (d *Deployer) Mine() error {
	return d.rpc.Call(nil, "generate", 1)
}

func (d *Deployer) ConfirmContract(c *contract.DeployedContract) (err error) {
	for {
		// fmt.Printf("Checking %s\n", name)
		result := make(map[string]interface{})
		err := d.rpc.Call(&result, "getaccountinfo", c.Address)
		if err, ok := err.(*jsonRPCError); ok {
			// fmt.Printf("%s\t%s\n", name, err)
			nudge := rand.Intn(500)
			time.Sleep(1*time.Second + time.Duration(nudge)*time.Millisecond)
			continue
		} else if err != nil {
			return err
		}

		// fmt.Printf("confirmed\t%s\t%s\n", name, c.Address)
		c.Confirmed = true
		return nil
	}
}

func (d *Deployer) CreateContract(c *contract.CompiledContract, jsonParams []byte, name string, overwrite bool) (err error) {
	if !overwrite && d.ContractsRepository.Exists(name) {
		return errors.Errorf("name already used: %s", name)
	}

	gasLimit := 300000

	bin, err := c.ToBytes(jsonParams)
	if err != nil {
		return
	}

	var tx TransactionReceipt

	err = d.rpc.Call(&tx, "createcontract", bin, gasLimit)

	if err != nil {
		return errors.Wrap(err, "createcontract")
	}

	// fmt.Println("tx", tx.Address)
	// fmt.Println("contract name", contract.Name)

	deployedContract := &contract.DeployedContract{
		CompiledContract: *c,
		Name:             c.Name,
		DeployName:       name,
		TransactionID:    tx.TxID,
		Address:          tx.Address,
		CreatedAt:        time.Now(),
	}

	err = d.ContractsRepository.Set(name, deployedContract)
	if err != nil {
		return
	}

	err = d.ContractsRepository.Commit()
	if err != nil {
		return
	}

	return nil
}
