package deepmind

import (
	"fmt"
	"time"

	pbcosmos "github.com/figment-networks/proto-cosmos/pb/sf/cosmos/type/v1"
	"github.com/golang/protobuf/proto"
	abci "github.com/tendermint/tendermint/abci/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/proto/tendermint/crypto"
	"github.com/tendermint/tendermint/types"
)

func mapBlockID(bid types.BlockID) *pbcosmos.BlockID {
	return &pbcosmos.BlockID{
		Hash: bid.Hash,
		PartSetHeader: &pbcosmos.PartSetHeader{
			Total: bid.PartSetHeader.Total,
			Hash:  bid.PartSetHeader.Hash,
		},
	}
}

func mapProposer(val *types.Validator) *pbcosmos.Validator {
	nPK := &pbcosmos.PublicKey{}

	return &pbcosmos.Validator{
		Address:          val.Address,
		PubKey:           nPK,
		ProposerPriority: 0,
	}
}

func mapEvent(ev abci.Event) *pbcosmos.Event {
	cev := &pbcosmos.Event{EventType: ev.Type}

	for _, at := range ev.Attributes {
		cev.Attributes = append(cev.Attributes, &pbcosmos.EventAttribute{
			Key:   string(at.Key),
			Value: string(at.Value),
			Index: at.Index,
		})
	}

	return cev
}

func mapVote(edv *types.Vote) *pbcosmos.EventVote {
	return &pbcosmos.EventVote{
		EventVoteType:    pbcosmos.SignedMsgType(edv.Type),
		Height:           uint64(edv.Height),
		Round:            edv.Round,
		BlockId:          mapBlockID(edv.BlockID),
		Timestamp:        mapTimestamp(edv.Timestamp),
		ValidatorAddress: edv.ValidatorAddress,
		ValidatorIndex:   edv.ValidatorIndex,
		Signature:        edv.Signature,
	}
}

func mapSignatures(commitSignatures []types.CommitSig) ([]*pbcosmos.CommitSig, error) {
	signatures := make([]*pbcosmos.CommitSig, len(commitSignatures))
	for i, commitSignature := range commitSignatures {
		signature, err := mapSignature(commitSignature)
		if err != nil {
			return nil, err
		}
		signatures[i] = signature
	}
	return signatures, nil
}

func mapSignature(s types.CommitSig) (*pbcosmos.CommitSig, error) {
	return &pbcosmos.CommitSig{
		BlockIdFlag:      pbcosmos.BlockIDFlag(s.BlockIDFlag),
		ValidatorAddress: s.ValidatorAddress.Bytes(),
		Timestamp:        mapTimestamp(s.Timestamp),
		Signature:        s.Signature,
	}, nil
}

func mapValidatorUpdate(v abci.ValidatorUpdate) (*pbcosmos.ValidatorUpdate, error) {
	nPK := &pbcosmos.PublicKey{}
	var address []byte

	switch key := v.PubKey.Sum.(type) {
	case *crypto.PublicKey_Ed25519:
		nPK.Sum = &pbcosmos.PublicKey_Ed25519{Ed25519: key.Ed25519}
		address = tmcrypto.AddressHash(nPK.GetEd25519())
	case *crypto.PublicKey_Secp256K1:
		nPK.Sum = &pbcosmos.PublicKey_Secp256K1{Secp256K1: key.Secp256K1}
		address = tmcrypto.AddressHash(nPK.GetSecp256K1())
	default:
		return nil, fmt.Errorf("given type %T of PubKey mapping doesn't exist ", key)
	}

	return &pbcosmos.ValidatorUpdate{
		Address: address,
		PubKey:  nPK,
		Power:   v.Power,
	}, nil
}

func mapValidators(srcValidators []*types.Validator) ([]*pbcosmos.Validator, error) {
	validators := make([]*pbcosmos.Validator, len(srcValidators))
	for i, validator := range srcValidators {
		val, err := mapValidator(validator)
		if err != nil {
			return nil, err
		}
		validators[i] = val
	}
	return validators, nil
}

func mapValidator(v *types.Validator) (*pbcosmos.Validator, error) {
	nPK := &pbcosmos.PublicKey{}

	key := v.PubKey
	switch key.Type() {
	case ed25519.KeyType:
		nPK = &pbcosmos.PublicKey{
			Sum: &pbcosmos.PublicKey_Ed25519{Ed25519: key.Bytes()}}
	case secp256k1.KeyType:
		nPK = &pbcosmos.PublicKey{
			Sum: &pbcosmos.PublicKey_Secp256K1{Secp256K1: key.Bytes()}}
	default:
		return nil, fmt.Errorf("given type %T of PubKey mapping doesn't exist ", key)
	}

	// NOTE: See note in mapValidatorUpdate() about ProposerPriority

	return &pbcosmos.Validator{
		Address:          v.Address,
		PubKey:           nPK,
		VotingPower:      v.VotingPower,
		ProposerPriority: 0,
	}, nil
}

func mapTimestamp(time time.Time) *pbcosmos.Timestamp {
	return &pbcosmos.Timestamp{
		Seconds: time.Unix(),
		Nanos:   int32(time.UnixNano() - time.Unix()*1000000000),
	}
}

func mapTx(bytes []byte) (*pbcosmos.Tx, error) {
	t := &pbcosmos.Tx{}
	if err := proto.Unmarshal(bytes, t); err != nil {
		return nil, err
	}
	return t, nil
}
