package deepmind

import (
	"fmt"

	pbcosmos "github.com/figment-networks/proto-cosmos/pb/sf/cosmos/type/v1"
	"github.com/golang/protobuf/proto"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
	"github.com/tendermint/tendermint/types"
)

func encodeBlock(bh types.EventDataNewBlock) ([]byte, error) {
	mappedCommitSignatures, err := mapSignatures(bh.Block.LastCommit.Signatures)
	if err != nil {
		return nil, err
	}

	nb := &pbcosmos.Block{
		Header: &pbcosmos.Header{
			Version: &pbcosmos.Consensus{
				Block: bh.Block.Header.Version.Block,
				App:   bh.Block.Header.Version.App,
			},
			ChainId:            bh.Block.Header.ChainID,
			Height:             uint64(bh.Block.Header.Height),
			Time:               mapTimestamp(bh.Block.Header.Time),
			LastBlockId:        mapBlockID(bh.Block.LastBlockID),
			LastCommitHash:     bh.Block.Header.LastCommitHash,
			DataHash:           bh.Block.Header.DataHash,
			ValidatorsHash:     bh.Block.Header.ValidatorsHash,
			NextValidatorsHash: bh.Block.Header.NextValidatorsHash,
			ConsensusHash:      bh.Block.Header.ConsensusHash,
			AppHash:            bh.Block.Header.AppHash,
			LastResultsHash:    bh.Block.Header.LastResultsHash,
			EvidenceHash:       bh.Block.Header.EvidenceHash,
			ProposerAddress:    bh.Block.Header.ProposerAddress,
			Hash:               bh.Block.Header.Hash(),
		},
		LastCommit: &pbcosmos.Commit{
			Height:     bh.Block.LastCommit.Height,
			Round:      bh.Block.LastCommit.Round,
			BlockId:    mapBlockID(bh.Block.LastCommit.BlockID),
			Signatures: mappedCommitSignatures,
		},
		Evidence: &pbcosmos.EvidenceList{},
	}

	if len(bh.Block.Evidence.Evidence) > 0 {
		for _, ev := range bh.Block.Evidence.Evidence {

			newEv := &pbcosmos.Evidence{}
			switch evN := ev.(type) {
			case *types.DuplicateVoteEvidence:
				newEv.Sum = &pbcosmos.Evidence_DuplicateVoteEvidence{
					DuplicateVoteEvidence: &pbcosmos.DuplicateVoteEvidence{
						VoteA:            mapVote(evN.VoteA),
						VoteB:            mapVote(evN.VoteB),
						TotalVotingPower: evN.TotalVotingPower,
						ValidatorPower:   evN.ValidatorPower,
						Timestamp:        mapTimestamp(evN.Timestamp),
					},
				}
			case *types.LightClientAttackEvidence:
				mappedSetValidators, err := mapValidators(evN.ConflictingBlock.ValidatorSet.Validators)
				if err != nil {
					return nil, err
				}

				mappedByzantineValidators, err := mapValidators(evN.ByzantineValidators)
				if err != nil {
					return nil, err
				}

				mappedCommitSignatures, err := mapSignatures(evN.ConflictingBlock.Commit.Signatures)
				if err != nil {
					return nil, err
				}

				newEv.Sum = &pbcosmos.Evidence_LightClientAttackEvidence{
					LightClientAttackEvidence: &pbcosmos.LightClientAttackEvidence{
						ConflictingBlock: &pbcosmos.LightBlock{
							SignedHeader: &pbcosmos.SignedHeader{
								Header: &pbcosmos.Header{
									Version: &pbcosmos.Consensus{
										Block: evN.ConflictingBlock.Version.Block,
										App:   evN.ConflictingBlock.Version.App,
									},
									ChainId:            evN.ConflictingBlock.Header.ChainID,
									Height:             uint64(evN.ConflictingBlock.Header.Height),
									Time:               mapTimestamp(evN.ConflictingBlock.Header.Time),
									LastBlockId:        mapBlockID(evN.ConflictingBlock.Header.LastBlockID),
									LastCommitHash:     evN.ConflictingBlock.Header.LastCommitHash,
									DataHash:           evN.ConflictingBlock.Header.DataHash,
									ValidatorsHash:     evN.ConflictingBlock.Header.ValidatorsHash,
									NextValidatorsHash: evN.ConflictingBlock.Header.NextValidatorsHash,
									ConsensusHash:      evN.ConflictingBlock.Header.ConsensusHash,
									AppHash:            evN.ConflictingBlock.Header.AppHash,
									LastResultsHash:    evN.ConflictingBlock.Header.LastResultsHash,
									EvidenceHash:       evN.ConflictingBlock.Header.EvidenceHash,
									ProposerAddress:    evN.ConflictingBlock.Header.ProposerAddress,
								},
								Commit: &pbcosmos.Commit{
									Height:     evN.ConflictingBlock.Commit.Height,
									Round:      evN.ConflictingBlock.Commit.Round,
									BlockId:    mapBlockID(evN.ConflictingBlock.Commit.BlockID),
									Signatures: mappedCommitSignatures,
								},
							},
							ValidatorSet: &pbcosmos.ValidatorSet{
								Validators:       mappedSetValidators,
								Proposer:         mapProposer(evN.ConflictingBlock.ValidatorSet.Proposer),
								TotalVotingPower: evN.ConflictingBlock.ValidatorSet.TotalVotingPower(),
							},
						},
						CommonHeight:        evN.CommonHeight,
						ByzantineValidators: mappedByzantineValidators,
						TotalVotingPower:    evN.TotalVotingPower,
						Timestamp:           mapTimestamp(evN.Timestamp),
					},
				}

			default:
				return nil, fmt.Errorf("given type %T of EvidenceList mapping doesn't exist ", ev)
			}

			nb.Evidence.Evidence = append(nb.Evidence.Evidence, newEv)
		}
	}

	if len(bh.ResultBeginBlock.Events) > 0 {
		nb.ResultBeginBlock = &pbcosmos.ResponseBeginBlock{}
		for _, ev := range bh.ResultBeginBlock.Events {
			nb.ResultBeginBlock.Events = append(nb.ResultBeginBlock.Events, mapEvent(ev))
		}
	}

	if len(bh.ResultEndBlock.Events) > 0 || len(bh.ResultEndBlock.ValidatorUpdates) > 0 || bh.ResultEndBlock.ConsensusParamUpdates != nil {
		nb.ResultEndBlock = &pbcosmos.ResponseEndBlock{
			ConsensusParamUpdates: &pbcosmos.ConsensusParams{},
		}

		for _, ev := range bh.ResultEndBlock.Events {
			nb.ResultEndBlock.Events = append(nb.ResultEndBlock.Events, mapEvent(ev))
		}

		for _, v := range bh.ResultEndBlock.ValidatorUpdates {
			val, err := mapValidatorUpdate(v)
			if err != nil {
				return nil, err
			}
			nb.ResultEndBlock.ValidatorUpdates = append(nb.ResultEndBlock.ValidatorUpdates, val)
		}
	}

	return proto.Marshal(nb)
}

func encodeTx(result *abci.TxResult) ([]byte, error) {
	mappedTx, err := mapTx(result.Tx)
	if err != nil {
		return nil, err
	}

	tx := &pbcosmos.TxResult{
		Hash:   tmhash.Sum(result.Tx),
		Height: uint64(result.Height),
		Index:  result.Index,
		Tx:     mappedTx,
		Result: &pbcosmos.ResponseDeliverTx{
			Code:      result.Result.Code,
			Data:      result.Result.Data,
			Log:       result.Result.Log,
			Info:      result.Result.Info,
			GasWanted: result.Result.GasWanted,
			GasUsed:   result.Result.GasUsed,
			Codespace: result.Result.Codespace,
		},
	}

	for _, ev := range result.Result.Events {
		tx.Result.Events = append(tx.Result.Events, mapEvent(ev))
	}

	return proto.Marshal(tx)
}

func encodeValidatorSetUpdates(updates *types.EventDataValidatorSetUpdates) ([]byte, error) {
	result := &pbcosmos.ValidatorSetUpdates{}

	for _, update := range updates.ValidatorUpdates {
		nPK := &pbcosmos.PublicKey{}

		switch update.PubKey.Type() {
		case "ed25519":
			nPK.Sum = &pbcosmos.PublicKey_Ed25519{Ed25519: update.PubKey.Bytes()}
		case "secp256k1":
			nPK.Sum = &pbcosmos.PublicKey_Secp256K1{Secp256K1: update.PubKey.Bytes()}
		default:
			return nil, fmt.Errorf("unsupported pubkey type: %T", update.PubKey)
		}

		result.ValidatorUpdates = append(result.ValidatorUpdates, &pbcosmos.Validator{
			Address:          update.Address.Bytes(),
			VotingPower:      update.VotingPower,
			ProposerPriority: update.ProposerPriority,
			PubKey:           nPK,
		})
	}

	return proto.Marshal(result)
}
