package controller

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type mockSQSClientType struct {
	mu       sync.Mutex
	messages []sqstypes.Message
}

var _ SQSClient = &mockSQSClientType{}

func (s *mockSQSClientType) append(msg sqstypes.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msg)
}

func (s *mockSQSClientType) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return &sqs.ReceiveMessageOutput{Messages: s.messages}, nil
}

func (s *mockSQSClientType) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(options *sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = nil
	return &sqs.DeleteMessageOutput{}, nil
}
