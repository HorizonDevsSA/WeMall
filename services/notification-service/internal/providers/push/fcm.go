package push

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/rs/zerolog"
	"google.golang.org/api/option"
)

type FCMProvider struct {
	client *messaging.Client
	logger zerolog.Logger
}

func NewFCMProvider(credJSON, credPath string, logger zerolog.Logger) (*FCMProvider, error) {
	// If credentials are empty, run in stub/mock mode
	if credJSON == "" && credPath == "" {
		logger.Info().Msg("[STUB MODE] FCM Provider initialized in mock mode (no Firebase credentials)")
		return &FCMProvider{client: nil, logger: logger}, nil
	}

	var opt option.ClientOption
	if credJSON != "" {
		opt = option.WithCredentialsJSON([]byte(credJSON))
	} else {
		opt = option.WithCredentialsFile(credPath)
	}

	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("firebase new app: %w", err)
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		return nil, fmt.Errorf("firebase messaging client: %w", err)
	}

	return &FCMProvider{
		client: client,
		logger: logger,
	}, nil
}

type PushMessage struct {
	Token        string
	Title        string
	Body         string
	Data         map[string]string
	Category     string
}

func (p *FCMProvider) SendPush(ctx context.Context, msg *PushMessage) (string, error) {
	if p.client == nil {
		p.logger.Info().
			Str("token", msg.Token).
			Str("title", msg.Title).
			Str("body", msg.Body).
			Msg("[STUB MODE] Push notification simulated successfully")
		return "mock-message-id", nil
	}

	fcmMsg := &messaging.Message{
		Token: msg.Token,
		Notification: &messaging.Notification{
			Title: msg.Title,
			Body:  msg.Body,
		},
		Data: msg.Data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				ChannelID: "wemall_channel_01",
				Color:     "#6c63ff",
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: msg.Title,
						Body:  msg.Body,
					},
					Badge: intPtr(1),
					Sound: "default",
				},
			},
		},
	}

	msgID, err := p.client.Send(ctx, fcmMsg)
	if err != nil {
		return "", err
	}
	return msgID, nil
}

// SendMulticast sends a push to multiple tokens in batches of 500.
// Returns a list of tokens that failed with "Unregistered" or "InvalidArgument" so they can be pruned.
func (p *FCMProvider) SendMulticast(ctx context.Context, tokens []string, title, body string, data map[string]string) ([]string, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	if p.client == nil {
		p.logger.Info().
			Int("tokens_count", len(tokens)).
			Str("title", title).
			Str("body", body).
			Msg("[STUB MODE] Multicast push notifications simulated successfully")
		// Simulate token pruning for test tokens starting with "invalid_"
		var invalidTokens []string
		for _, t := range tokens {
			if len(t) > 8 && t[:8] == "invalid_" {
				invalidTokens = append(invalidTokens, t)
			}
		}
		return invalidTokens, nil
	}

	var failedTokens []string
	batchSize := 500

	for i := 0; i < len(tokens); i += batchSize {
		end := i + batchSize
		if end > len(tokens) {
			end = len(tokens)
		}
		batch := tokens[i:end]

		multicastMsg := &messaging.MulticastMessage{
			Tokens: batch,
			Notification: &messaging.Notification{
				Title: title,
				Body:  body,
			},
			Data: data,
			Android: &messaging.AndroidConfig{
				Priority: "high",
				Notification: &messaging.AndroidNotification{
					ChannelID: "wemall_channel_01",
					Color:     "#6c63ff",
				},
			},
			APNS: &messaging.APNSConfig{
				Payload: &messaging.APNSPayload{
					Aps: &messaging.Aps{
						Alert: &messaging.ApsAlert{
							Title: title,
							Body:  body,
						},
						Badge: intPtr(1),
						Sound: "default",
					},
				},
			},
		}

		response, err := p.client.SendEachForMulticast(ctx, multicastMsg)
		if err != nil {
			return nil, fmt.Errorf("multicast batch send: %w", err)
		}

		for idx, resp := range response.Responses {
			if !resp.Success {
				token := batch[idx]
				if messaging.IsUnregistered(resp.Error) || messaging.IsInvalidArgument(resp.Error) {
					failedTokens = append(failedTokens, token)
				}
				p.logger.Error().Err(resp.Error).Str("token", token).Msg("Failed to send push notification to token")
			}
		}
	}

	return failedTokens, nil
}

func intPtr(i int) *int {
	return &i
}
