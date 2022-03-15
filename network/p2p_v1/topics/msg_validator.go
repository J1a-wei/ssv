package topics

import (
	"bytes"
	"context"
	forksv1 "github.com/bloxapp/ssv/network/forks/v1"
	forks2 "github.com/bloxapp/ssv/operator/forks"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"go.uber.org/zap"
)

// MsgValidatorFunc represents a message validator
type MsgValidatorFunc = func(ctx context.Context, p peer.ID, msg *pubsub.Message) pubsub.ValidationResult

// NewSSVMsgValidator creates a new msg validator that validates message structure,
// and checks that the message was sent on the right topic.
// TODO: remove logs, break into smaller validators?
func NewSSVMsgValidator(plogger *zap.Logger, fork *forks2.Forker, self peer.ID) func(ctx context.Context, p peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	return func(ctx context.Context, p peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
		logger := plogger.With(zap.String("topic", msg.GetTopic()), zap.String("peer", p.String()))
		//logger.Debug("xxx validating msg")
		if len(msg.GetData()) == 0 {
			logger.Debug("invalid: no data")
			reportValidationResult(validationResultNoData)
			return pubsub.ValidationReject
		}
		if bytes.Equal([]byte(p), []byte(self)) {
			logger.Debug("valid:  our node's message")
			reportValidationResult(validationResultSelf)
			return pubsub.ValidationAccept
		}
		smsg, err := fork.GetCurrentFork().NetworkFork().(*forksv1.ForkV1).DecodeNetworkMsgV1(msg.GetData())
		if err != nil {
			// can't decode message
			logger.Debug("invalid: can't decode message", zap.Error(err))
			reportValidationResult(validationResultEncoding)
			return pubsub.ValidationReject
		}
		topic := fork.GetCurrentFork().NetworkFork().ValidatorTopicID(smsg.ID.GetValidatorPK())
		if topic != msg.GetTopic() {
			// wrong topic
			logger.Debug("invalid: wrong topic", zap.String("actual", topic),
				zap.String("expected", msg.GetTopic()),
				zap.ByteString("smsg.ID", smsg.ID))
			reportValidationResult(validationResultTopic)
			return pubsub.ValidationReject
		}
		logger.Debug("valid", zap.ByteString("smsg.ID", smsg.ID))
		reportValidationResult(validationResultValid)
		return pubsub.ValidationAccept
	}
}

//// CombineMsgValidators executes multiple validators
//func CombineMsgValidators(validators ...MsgValidatorFunc) MsgValidatorFunc {
//	return func(ctx context.Context, p peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
//		res := pubsub.ValidationAccept
//		for _, v := range validators {
//			if res = v(ctx, p, msg); res == pubsub.ValidationReject {
//				break
//			}
//		}
//		return res
//	}
//}
