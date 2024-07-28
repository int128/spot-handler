package awssqs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type Client interface {
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(options *sqs.Options)) (*sqs.DeleteMessageOutput, error)
}
