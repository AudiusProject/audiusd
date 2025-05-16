// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package gen

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// ProportionalRewardsRewardRatio is an auto generated low-level Go binding around an user-defined struct.
type ProportionalRewardsRewardRatio struct {
	Node  common.Address
	Ratio uint8
}

// ProportionalRewardsMetaData contains all meta data concerning the ProportionalRewards contract.
var ProportionalRewardsMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"roundNumber\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"submitter\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"submitterStake\",\"type\":\"uint256\"}],\"name\":\"HashSubmitted\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"stakingContractAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"roundNumber\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"node\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"ratio\",\"type\":\"uint8\"}],\"internalType\":\"structProportionalRewards.RewardRatio[]\",\"name\":\"ratios\",\"type\":\"tuple[]\"}],\"name\":\"finalizeProposal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"stakingContractAddress\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"roundNumber\",\"type\":\"uint256\"}],\"name\":\"submitSummaryHash\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b50610f6b8061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c80634134981214610038578063d674e67f14610054575b5f5ffd5b610052600480360381019061004d9190610725565b610070565b005b61006e600480360381019061006991906107d6565b610381565b005b5f61007a82610654565b90505f3390505f8573ffffffffffffffffffffffffffffffffffffffff1663ede3842183856040518363ffffffff1660e01b81526004016100bc929190610865565b602060405180830381865afa1580156100d7573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906100fb91906108a0565b90505f811161013f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161013690610971565b60405180910390fd5b5f5f5f8681526020019081526020015f206002015f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205490505f60018111156101a4576101a361098f565b5b5f5f8781526020019081526020015f206001015f9054906101000a900460ff1660018111156101d6576101d561098f565b5b14806101e357505f5f1b81145b610222576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161021990610a52565b60405180910390fd5b5f5f1b811461028057815f5f8781526020019081526020015f206003015f8381526020019081526020015f20546102599190610a9d565b5f5f8781526020019081526020015f206003015f8381526020019081526020015f20819055505b855f5f8781526020019081526020015f206002015f8573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f2081905550815f5f8781526020019081526020015f206003015f8881526020019081526020015f20546103019190610ad0565b5f5f8781526020019081526020015f206003015f8881526020019081526020015f20819055508273ffffffffffffffffffffffffffffffffffffffff16857fefb6cf1c6b69b30a849971f578a6fe51fa823a0ecd3cd5c40f3396dfd81fad658885604051610370929190610b12565b60405180910390a350505050505050565b6001808111156103945761039361098f565b5b5f5f8581526020019081526020015f206001015f9054906101000a900460ff1660018111156103c6576103c561098f565b5b03610406576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016103fd90610ba9565b60405180910390fd5b5f61041084610654565b90505f8573ffffffffffffffffffffffffffffffffffffffff1663c9c53232836040518263ffffffff1660e01b815260040161044c9190610bc7565b602060405180830381865afa158015610467573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061048b91906108a0565b90505f84846040516020016104a1929190610d3e565b60405160208183030381529060405280519060200120905060428260645f5f8a81526020019081526020015f206003015f8581526020019081526020015f20546104eb9190610d60565b6104f59190610dce565b11610535576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161052c90610e94565b60405180910390fd5b5f8585905090505f5f90505b8181101561060e5786868281811061055c5761055b610eb2565b5b90506040020160200160208101906105749190610edf565b5f5f8a81526020019081526020015f205f015f89898581811061059a57610599610eb2565b5b9050604002015f0160208101906105b19190610f0a565b73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f6101000a81548160ff021916908360ff1602179055508080600101915050610541565b5060015f5f8981526020019081526020015f206001015f6101000a81548160ff021916908360018111156106455761064461098f565b5b02179055505050505050505050565b5f439050919050565b5f5ffd5b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61068e82610665565b9050919050565b61069e81610684565b81146106a8575f5ffd5b50565b5f813590506106b981610695565b92915050565b5f819050919050565b6106d1816106bf565b81146106db575f5ffd5b50565b5f813590506106ec816106c8565b92915050565b5f819050919050565b610704816106f2565b811461070e575f5ffd5b50565b5f8135905061071f816106fb565b92915050565b5f5f5f6060848603121561073c5761073b61065d565b5b5f610749868287016106ab565b935050602061075a868287016106de565b925050604061076b86828701610711565b9150509250925092565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f84011261079657610795610775565b5b8235905067ffffffffffffffff8111156107b3576107b2610779565b5b6020830191508360408202830111156107cf576107ce61077d565b5b9250929050565b5f5f5f5f606085870312156107ee576107ed61065d565b5b5f6107fb878288016106ab565b945050602061080c87828801610711565b935050604085013567ffffffffffffffff81111561082d5761082c610661565b5b61083987828801610781565b925092505092959194509250565b61085081610684565b82525050565b61085f816106f2565b82525050565b5f6040820190506108785f830185610847565b6108856020830184610856565b9392505050565b5f8151905061089a816106fb565b92915050565b5f602082840312156108b5576108b461065d565b5b5f6108c28482850161088c565b91505092915050565b5f82825260208201905092915050565b7f50726f706f7274696f6e616c526577617264733a207375626d6974746572206d5f8201527f75737420626520616464726573732077697468206e6f6e2d7a65726f2073746160208201527f6b652e0000000000000000000000000000000000000000000000000000000000604082015250565b5f61095b6043836108cb565b9150610966826108db565b606082019050919050565b5f6020820190508181035f8301526109888161094f565b9050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602160045260245ffd5b7f50726f706f7274696f6e616c526577617264733a2063616e6e6f74206368616e5f8201527f676520766f746520616674657220726577617264732070726f706f73616c206960208201527f732066696e616c697a6564000000000000000000000000000000000000000000604082015250565b5f610a3c604b836108cb565b9150610a47826109bc565b606082019050919050565b5f6020820190508181035f830152610a6981610a30565b9050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f610aa7826106f2565b9150610ab2836106f2565b9250828203905081811115610aca57610ac9610a70565b5b92915050565b5f610ada826106f2565b9150610ae5836106f2565b9250828201905080821115610afd57610afc610a70565b5b92915050565b610b0c816106bf565b82525050565b5f604082019050610b255f830185610b03565b610b326020830184610856565b9392505050565b7f50726f706f7274696f6e616c526577617264733a20726f756e6420697320616c5f8201527f72656164792066696e616c697a65640000000000000000000000000000000000602082015250565b5f610b93602f836108cb565b9150610b9e82610b39565b604082019050919050565b5f6020820190508181035f830152610bc081610b87565b9050919050565b5f602082019050610bda5f830184610856565b92915050565b5f82825260208201905092915050565b5f819050919050565b5f610c0760208401846106ab565b905092915050565b610c1881610684565b82525050565b5f60ff82169050919050565b610c3381610c1e565b8114610c3d575f5ffd5b50565b5f81359050610c4e81610c2a565b92915050565b5f610c626020840184610c40565b905092915050565b610c7381610c1e565b82525050565b60408201610c895f830183610bf9565b610c955f850182610c0f565b50610ca36020830183610c54565b610cb06020850182610c6a565b50505050565b5f610cc18383610c79565b60408301905092915050565b5f82905092915050565b5f604082019050919050565b5f610cee8385610be0565b9350610cf982610bf0565b805f5b85811015610d3157610d0e8284610ccd565b610d188882610cb6565b9750610d2383610cd7565b925050600181019050610cfc565b5085925050509392505050565b5f6020820190508181035f830152610d57818486610ce3565b90509392505050565b5f610d6a826106f2565b9150610d75836106f2565b9250828202610d83816106f2565b91508282048414831517610d9a57610d99610a70565b5b5092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601260045260245ffd5b5f610dd8826106f2565b9150610de3836106f2565b925082610df357610df2610da1565b5b828204905092915050565b7f50726f706f7274696f6e616c526577617264733a20496e73756666696369656e5f8201527f7420636f6e73656e73757320666f722070726f706f736564207265776172642060208201527f726174696f730000000000000000000000000000000000000000000000000000604082015250565b5f610e7e6046836108cb565b9150610e8982610dfe565b606082019050919050565b5f6020820190508181035f830152610eab81610e72565b9050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b5f60208284031215610ef457610ef361065d565b5b5f610f0184828501610c40565b91505092915050565b5f60208284031215610f1f57610f1e61065d565b5b5f610f2c848285016106ab565b9150509291505056fea2646970667358221220f01ccd09600aa4c93a925a4386855795b19d28c1643d140d96d09f54edfc009d64736f6c634300081e0033",
}

// ProportionalRewardsABI is the input ABI used to generate the binding from.
// Deprecated: Use ProportionalRewardsMetaData.ABI instead.
var ProportionalRewardsABI = ProportionalRewardsMetaData.ABI

// ProportionalRewardsBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ProportionalRewardsMetaData.Bin instead.
var ProportionalRewardsBin = ProportionalRewardsMetaData.Bin

// DeployProportionalRewards deploys a new Ethereum contract, binding an instance of ProportionalRewards to it.
func DeployProportionalRewards(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ProportionalRewards, error) {
	parsed, err := ProportionalRewardsMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ProportionalRewardsBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ProportionalRewards{ProportionalRewardsCaller: ProportionalRewardsCaller{contract: contract}, ProportionalRewardsTransactor: ProportionalRewardsTransactor{contract: contract}, ProportionalRewardsFilterer: ProportionalRewardsFilterer{contract: contract}}, nil
}

// ProportionalRewards is an auto generated Go binding around an Ethereum contract.
type ProportionalRewards struct {
	ProportionalRewardsCaller     // Read-only binding to the contract
	ProportionalRewardsTransactor // Write-only binding to the contract
	ProportionalRewardsFilterer   // Log filterer for contract events
}

// ProportionalRewardsCaller is an auto generated read-only Go binding around an Ethereum contract.
type ProportionalRewardsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProportionalRewardsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ProportionalRewardsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProportionalRewardsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ProportionalRewardsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProportionalRewardsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ProportionalRewardsSession struct {
	Contract     *ProportionalRewards // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// ProportionalRewardsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ProportionalRewardsCallerSession struct {
	Contract *ProportionalRewardsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// ProportionalRewardsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ProportionalRewardsTransactorSession struct {
	Contract     *ProportionalRewardsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// ProportionalRewardsRaw is an auto generated low-level Go binding around an Ethereum contract.
type ProportionalRewardsRaw struct {
	Contract *ProportionalRewards // Generic contract binding to access the raw methods on
}

// ProportionalRewardsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ProportionalRewardsCallerRaw struct {
	Contract *ProportionalRewardsCaller // Generic read-only contract binding to access the raw methods on
}

// ProportionalRewardsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ProportionalRewardsTransactorRaw struct {
	Contract *ProportionalRewardsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewProportionalRewards creates a new instance of ProportionalRewards, bound to a specific deployed contract.
func NewProportionalRewards(address common.Address, backend bind.ContractBackend) (*ProportionalRewards, error) {
	contract, err := bindProportionalRewards(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ProportionalRewards{ProportionalRewardsCaller: ProportionalRewardsCaller{contract: contract}, ProportionalRewardsTransactor: ProportionalRewardsTransactor{contract: contract}, ProportionalRewardsFilterer: ProportionalRewardsFilterer{contract: contract}}, nil
}

// NewProportionalRewardsCaller creates a new read-only instance of ProportionalRewards, bound to a specific deployed contract.
func NewProportionalRewardsCaller(address common.Address, caller bind.ContractCaller) (*ProportionalRewardsCaller, error) {
	contract, err := bindProportionalRewards(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ProportionalRewardsCaller{contract: contract}, nil
}

// NewProportionalRewardsTransactor creates a new write-only instance of ProportionalRewards, bound to a specific deployed contract.
func NewProportionalRewardsTransactor(address common.Address, transactor bind.ContractTransactor) (*ProportionalRewardsTransactor, error) {
	contract, err := bindProportionalRewards(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ProportionalRewardsTransactor{contract: contract}, nil
}

// NewProportionalRewardsFilterer creates a new log filterer instance of ProportionalRewards, bound to a specific deployed contract.
func NewProportionalRewardsFilterer(address common.Address, filterer bind.ContractFilterer) (*ProportionalRewardsFilterer, error) {
	contract, err := bindProportionalRewards(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ProportionalRewardsFilterer{contract: contract}, nil
}

// bindProportionalRewards binds a generic wrapper to an already deployed contract.
func bindProportionalRewards(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ProportionalRewardsMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ProportionalRewards *ProportionalRewardsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ProportionalRewards.Contract.ProportionalRewardsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ProportionalRewards *ProportionalRewardsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ProportionalRewards.Contract.ProportionalRewardsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ProportionalRewards *ProportionalRewardsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ProportionalRewards.Contract.ProportionalRewardsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ProportionalRewards *ProportionalRewardsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ProportionalRewards.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ProportionalRewards *ProportionalRewardsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ProportionalRewards.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ProportionalRewards *ProportionalRewardsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ProportionalRewards.Contract.contract.Transact(opts, method, params...)
}

// FinalizeProposal is a paid mutator transaction binding the contract method 0xd674e67f.
//
// Solidity: function finalizeProposal(address stakingContractAddress, uint256 roundNumber, (address,uint8)[] ratios) returns()
func (_ProportionalRewards *ProportionalRewardsTransactor) FinalizeProposal(opts *bind.TransactOpts, stakingContractAddress common.Address, roundNumber *big.Int, ratios []ProportionalRewardsRewardRatio) (*types.Transaction, error) {
	return _ProportionalRewards.contract.Transact(opts, "finalizeProposal", stakingContractAddress, roundNumber, ratios)
}

// FinalizeProposal is a paid mutator transaction binding the contract method 0xd674e67f.
//
// Solidity: function finalizeProposal(address stakingContractAddress, uint256 roundNumber, (address,uint8)[] ratios) returns()
func (_ProportionalRewards *ProportionalRewardsSession) FinalizeProposal(stakingContractAddress common.Address, roundNumber *big.Int, ratios []ProportionalRewardsRewardRatio) (*types.Transaction, error) {
	return _ProportionalRewards.Contract.FinalizeProposal(&_ProportionalRewards.TransactOpts, stakingContractAddress, roundNumber, ratios)
}

// FinalizeProposal is a paid mutator transaction binding the contract method 0xd674e67f.
//
// Solidity: function finalizeProposal(address stakingContractAddress, uint256 roundNumber, (address,uint8)[] ratios) returns()
func (_ProportionalRewards *ProportionalRewardsTransactorSession) FinalizeProposal(stakingContractAddress common.Address, roundNumber *big.Int, ratios []ProportionalRewardsRewardRatio) (*types.Transaction, error) {
	return _ProportionalRewards.Contract.FinalizeProposal(&_ProportionalRewards.TransactOpts, stakingContractAddress, roundNumber, ratios)
}

// SubmitSummaryHash is a paid mutator transaction binding the contract method 0x41349812.
//
// Solidity: function submitSummaryHash(address stakingContractAddress, bytes32 hash, uint256 roundNumber) returns()
func (_ProportionalRewards *ProportionalRewardsTransactor) SubmitSummaryHash(opts *bind.TransactOpts, stakingContractAddress common.Address, hash [32]byte, roundNumber *big.Int) (*types.Transaction, error) {
	return _ProportionalRewards.contract.Transact(opts, "submitSummaryHash", stakingContractAddress, hash, roundNumber)
}

// SubmitSummaryHash is a paid mutator transaction binding the contract method 0x41349812.
//
// Solidity: function submitSummaryHash(address stakingContractAddress, bytes32 hash, uint256 roundNumber) returns()
func (_ProportionalRewards *ProportionalRewardsSession) SubmitSummaryHash(stakingContractAddress common.Address, hash [32]byte, roundNumber *big.Int) (*types.Transaction, error) {
	return _ProportionalRewards.Contract.SubmitSummaryHash(&_ProportionalRewards.TransactOpts, stakingContractAddress, hash, roundNumber)
}

// SubmitSummaryHash is a paid mutator transaction binding the contract method 0x41349812.
//
// Solidity: function submitSummaryHash(address stakingContractAddress, bytes32 hash, uint256 roundNumber) returns()
func (_ProportionalRewards *ProportionalRewardsTransactorSession) SubmitSummaryHash(stakingContractAddress common.Address, hash [32]byte, roundNumber *big.Int) (*types.Transaction, error) {
	return _ProportionalRewards.Contract.SubmitSummaryHash(&_ProportionalRewards.TransactOpts, stakingContractAddress, hash, roundNumber)
}

// ProportionalRewardsHashSubmittedIterator is returned from FilterHashSubmitted and is used to iterate over the raw logs and unpacked data for HashSubmitted events raised by the ProportionalRewards contract.
type ProportionalRewardsHashSubmittedIterator struct {
	Event *ProportionalRewardsHashSubmitted // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ProportionalRewardsHashSubmittedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ProportionalRewardsHashSubmitted)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ProportionalRewardsHashSubmitted)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ProportionalRewardsHashSubmittedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ProportionalRewardsHashSubmittedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ProportionalRewardsHashSubmitted represents a HashSubmitted event raised by the ProportionalRewards contract.
type ProportionalRewardsHashSubmitted struct {
	RoundNumber    *big.Int
	Submitter      common.Address
	Hash           [32]byte
	SubmitterStake *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterHashSubmitted is a free log retrieval operation binding the contract event 0xefb6cf1c6b69b30a849971f578a6fe51fa823a0ecd3cd5c40f3396dfd81fad65.
//
// Solidity: event HashSubmitted(uint256 indexed roundNumber, address indexed submitter, bytes32 hash, uint256 submitterStake)
func (_ProportionalRewards *ProportionalRewardsFilterer) FilterHashSubmitted(opts *bind.FilterOpts, roundNumber []*big.Int, submitter []common.Address) (*ProportionalRewardsHashSubmittedIterator, error) {

	var roundNumberRule []interface{}
	for _, roundNumberItem := range roundNumber {
		roundNumberRule = append(roundNumberRule, roundNumberItem)
	}
	var submitterRule []interface{}
	for _, submitterItem := range submitter {
		submitterRule = append(submitterRule, submitterItem)
	}

	logs, sub, err := _ProportionalRewards.contract.FilterLogs(opts, "HashSubmitted", roundNumberRule, submitterRule)
	if err != nil {
		return nil, err
	}
	return &ProportionalRewardsHashSubmittedIterator{contract: _ProportionalRewards.contract, event: "HashSubmitted", logs: logs, sub: sub}, nil
}

// WatchHashSubmitted is a free log subscription operation binding the contract event 0xefb6cf1c6b69b30a849971f578a6fe51fa823a0ecd3cd5c40f3396dfd81fad65.
//
// Solidity: event HashSubmitted(uint256 indexed roundNumber, address indexed submitter, bytes32 hash, uint256 submitterStake)
func (_ProportionalRewards *ProportionalRewardsFilterer) WatchHashSubmitted(opts *bind.WatchOpts, sink chan<- *ProportionalRewardsHashSubmitted, roundNumber []*big.Int, submitter []common.Address) (event.Subscription, error) {

	var roundNumberRule []interface{}
	for _, roundNumberItem := range roundNumber {
		roundNumberRule = append(roundNumberRule, roundNumberItem)
	}
	var submitterRule []interface{}
	for _, submitterItem := range submitter {
		submitterRule = append(submitterRule, submitterItem)
	}

	logs, sub, err := _ProportionalRewards.contract.WatchLogs(opts, "HashSubmitted", roundNumberRule, submitterRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ProportionalRewardsHashSubmitted)
				if err := _ProportionalRewards.contract.UnpackLog(event, "HashSubmitted", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseHashSubmitted is a log parse operation binding the contract event 0xefb6cf1c6b69b30a849971f578a6fe51fa823a0ecd3cd5c40f3396dfd81fad65.
//
// Solidity: event HashSubmitted(uint256 indexed roundNumber, address indexed submitter, bytes32 hash, uint256 submitterStake)
func (_ProportionalRewards *ProportionalRewardsFilterer) ParseHashSubmitted(log types.Log) (*ProportionalRewardsHashSubmitted, error) {
	event := new(ProportionalRewardsHashSubmitted)
	if err := _ProportionalRewards.contract.UnpackLog(event, "HashSubmitted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
