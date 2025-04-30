// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IStakingContract {
    function totalStakedForAt(address _accountAddress, uint256 _blockNumber) external view returns (uint256);
    function totalStakedAt(uint256 _blockNumber) external view returns (uint256);
}

contract ProportionalRewards {
    struct RewardRatio{ 
        address node;
        uint8 ratio;
    }

    enum Outcome {
        Undefined,
        Finalized
    }

    struct RewardsProposal {
        mapping(address => uint8) ratios;
        Outcome outcome;
        mapping(address => bytes32) submissions;
        mapping(bytes32 => uint256) submissionMagnitude;
    }

    event HashSubmitted(
        uint256 indexed roundNumber,
        address indexed submitter,
        bytes32 hash,
        uint256 submitterStake 
    );

    /// @notice mapping of roundNumber to (mapping of a round summary hash to the voting magnitude associated with that hash)
    mapping(uint256 => RewardsProposal) proposals;

	function submitSummaryHash(address stakingContractAddress, bytes32 hash, uint256 roundNumber) public {

        uint256 ethBlock = _getEthBlockForRound(roundNumber);
        address sp = msg.sender;

        // TODO: figure out difference between Staking.totalStakedForAt() and DelegateManager.getTotalDelegatedToServiceProvider()
        // TODO: stakingContractAddress should be initialized separately
        uint256 stake = IStakingContract(stakingContractAddress).totalStakedForAt(sp, ethBlock);

        // Require non-zero stake
        require(
            stake > 0,
            "ProportionalRewards: submitter must be address with non-zero stake."
        );

        bytes32 prevSubmission = proposals[roundNumber].submissions[sp];
        // Cannot change submission if proposal was finalized
        require(
            proposals[roundNumber].outcome == Outcome.Undefined || prevSubmission == bytes32(0),
            "ProportionalRewards: cannot change vote after rewards proposal is finalized"
        );

        // handle change of submission
        if (prevSubmission != bytes32(0)) {
            proposals[roundNumber].submissionMagnitude[prevSubmission] = proposals[roundNumber].submissionMagnitude[prevSubmission] - stake;
        }

        // record submission
        proposals[roundNumber].submissions[sp] = hash;

        // record submissionMagnitude
        proposals[roundNumber].submissionMagnitude[hash] = proposals[roundNumber].submissionMagnitude[hash] + stake;

        emit HashSubmitted(
            roundNumber,
            sp,
            hash,
            stake
        );
    }

	function finalizeProposal(address stakingContractAddress, uint256 roundNumber, RewardRatio[] calldata ratios) public {
        require(
            proposals[roundNumber].outcome != Outcome.Finalized,
            "ProportionalRewards: round is already finalized"
        );
        uint256 ethBlock = _getEthBlockForRound(roundNumber);
        // TODO: stakingContractAddress should be initialized separately
        uint256 totalStake = IStakingContract(stakingContractAddress).totalStakedAt(ethBlock);

        bytes32 ratioHash = keccak256(abi.encode(ratios));
        require(
            proposals[roundNumber].submissionMagnitude[ratioHash] * 100 / totalStake > 66,
            "ProportionalRewards: Insufficient consensus for proposed reward ratios"
        );
        uint256 len = ratios.length;
        for (uint256 i = 0; i < len; i++) {
            proposals[roundNumber].ratios[ratios[i].node] = ratios[i].ratio;
        }
        proposals[roundNumber].outcome = Outcome.Finalized;
    }

    function _getEthBlockForRound(uint256 roundNumber) internal view returns (uint256) {
        // TODO 
        return block.number;
    }
}
