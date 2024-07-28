package controller

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

var _ SQSClient = &mockSQSClientType{}

type mockSQSClientType struct {
	mu       sync.Mutex
	queueURL string
	messages []sqstypes.Message
}

func (s *mockSQSClientType) reset(queueURL string, messages []sqstypes.Message) {
	s.queueURL = queueURL
	s.messages = messages
}

func (s *mockSQSClientType) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	if aws.ToString(params.QueueUrl) != s.queueURL {
		return nil, fmt.Errorf("unknown queue %s", aws.ToString(params.QueueUrl))
	}
	return &sqs.ReceiveMessageOutput{Messages: s.messages}, nil
}

func (s *mockSQSClientType) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(options *sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if aws.ToString(params.QueueUrl) != s.queueURL {
		return nil, fmt.Errorf("unknown queue %s", aws.ToString(params.QueueUrl))
	}

	var found bool
	var newMessages []sqstypes.Message
	for _, message := range s.messages {
		if aws.ToString(message.ReceiptHandle) == aws.ToString(params.ReceiptHandle) {
			found = true
		} else {
			newMessages = append(newMessages, message)
		}
	}
	if !found {
		return nil, fmt.Errorf("no such message %s", aws.ToString(params.ReceiptHandle))
	}
	s.messages = newMessages
	return &sqs.DeleteMessageOutput{}, nil
}
