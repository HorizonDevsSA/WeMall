package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const (
	TypeEmailSend     = "email:send"
	TypePushSend      = "push:send"
	TypePushMulticast = "push:multicast"
)

type EmailSendPayload struct {
	UserID        string
	Category      string
	Recipient     string
	RecipientName string
	Subject       string
	HTMLBody      string
}

type PushSendPayload struct {
	UserID   string
	Category string
	Token    string
	Title    string
	Body     string
	Data     map[string]string
}

type PushMulticastPayload struct {
	UserIDs  []string
	Category string
	Tokens   []string
	Title    string
	Body     string
	Data     map[string]string
}

type Client struct {
	client *asynq.Client
}

func NewClient(redisAddr string) *Client {
	return &Client{
		client: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr}),
	}
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) EnqueueEmail(ctx context.Context, payload EmailSendPayload) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal email payload: %w", err)
	}

	task := asynq.NewTask(TypeEmailSend, bytes)
	_, err = c.client.EnqueueContext(ctx, task)
	return err
}

func (c *Client) EnqueuePush(ctx context.Context, payload PushSendPayload) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal push payload: %w", err)
	}

	task := asynq.NewTask(TypePushSend, bytes)
	_, err = c.client.EnqueueContext(ctx, task)
	return err
}

func (c *Client) EnqueuePushMulticast(ctx context.Context, payload PushMulticastPayload) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal multicast push payload: %w", err)
	}

	task := asynq.NewTask(TypePushMulticast, bytes)
	_, err = c.client.EnqueueContext(ctx, task)
	return err
}
