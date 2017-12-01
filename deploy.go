package solar

import (
	"fmt"

	"github.com/pkg/errors"
	"log"
)

func init() {
	cmd := app.Command("deploy", "Compile Solidity contracts.")

	force := cmd.Flag("force", "Overwrite previously deployed contract with the same name").Bool()
	noconfirm := cmd.Flag("no-confirm", "Don't wait for network to confirm deploy").Bool()
	noFastConfirm := cmd.Flag("no-fast-confirm", "(dev) Don't generate block to confirm deploy immediately").Bool()

	sourceFilePath := cmd.Arg("source", "Solidity contracts to deploy.").Required().String()
	name := cmd.Arg("name", "Name of contract").Required().String()
	jsonParams := cmd.Arg("jsonParams", "Constructor params as a json array").Default("").String()

	appTasks["deploy"] = func() (err error) {
		opts, err := solar.SolcOptions()
		if err != nil {
			return
		}

		filename := *sourceFilePath

		compiler := Compiler{
			Opts:     *opts,
			Filename: filename,
		}

		contract, err := compiler.Compile()
		if err != nil {
			return errors.Wrap(err, "compile")
		}

		deployer := solar.Deployer()
		repo := solar.ContractsRepository()

		fmt.Printf("   \033[36mdeploy\033[0m %s => %s\n", *sourceFilePath, *name)

		//fmt.Printf("%#v", deployer)
		err = deployer.CreateContract(contract, []byte(*jsonParams), *name, *force)
		if err != nil {
			fmt.Println("\u2757\ufe0f \033[36mdeploy\033[0m", err)
			return
		}

		newContracts := repo.UnconfirmedContracts()
		if *noconfirm == false && len(newContracts) != 0 {
			// Force local chain to generate a block immediately.
			allowFastConfirm := *solarEnv == "development" || *solarEnv == "test"
			if *noFastConfirm == false && allowFastConfirm {
				//fmt.Println("call deployer.Mine")
				err = deployer.Mine()
				if err != nil {
					log.Println(err)
				}
			}

			err := repo.ConfirmAll(getConfirmUpdateProgressFunc(), deployer.ConfirmContract)
			if err != nil {
				return err
			}
		}

		return
	}
}

