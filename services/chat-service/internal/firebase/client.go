package firebase

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

// FirestoreClient wraps a Cloud Firestore client for writing chat data.
type FirestoreClient struct {
	client *firestore.Client
}

// NewFirestoreClient initializes a Firestore client using the given service account JSON file.
func NewFirestoreClient(ctx context.Context, credentialsFile string) (*FirestoreClient, error) {
	opt := option.WithCredentialsFile(credentialsFile)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("firebase new app: %w", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("firestore client: %w", err)
	}

	return &FirestoreClient{client: client}, nil
}

// Close releases the Firestore client resources.
func (fc *FirestoreClient) Close() error {
	return fc.client.Close()
}

// EnsureThreadExists creates the thread document in Firestore if it doesn't already exist.
func (fc *FirestoreClient) EnsureThreadExists(ctx context.Context, threadID, threadType, title, sellerID string, members []string) error {
	docRef := fc.client.Collection("threads").Doc(threadID)
	doc, err := docRef.Get(ctx)
	if err == nil && doc.Exists() {
		return nil // already exists
	}

	data := map[string]interface{}{
		"type":      threadType,
		"title":     title,
		"sellerId":  sellerID,
		"members":   members,
		"createdAt": time.Now(),
		"updatedAt": time.Now(),
	}

	_, err = docRef.Set(ctx, data)
	if err != nil {
		return fmt.Errorf("create thread doc: %w", err)
	}
	log.Printf("[firestore] Created thread document %s", threadID)
	return nil
}

// AddMemberToThread appends a user ID to the thread's members array if not already present.
func (fc *FirestoreClient) AddMemberToThread(ctx context.Context, threadID, userID string) error {
	docRef := fc.client.Collection("threads").Doc(threadID)
	_, err := docRef.Update(ctx, []firestore.Update{
		{Path: "members", Value: firestore.ArrayUnion(userID)},
		{Path: "updatedAt", Value: time.Now()},
	})
	return err
}

// RemoveMemberFromThread removes a user ID from the thread's members array.
func (fc *FirestoreClient) RemoveMemberFromThread(ctx context.Context, threadID, userID string) error {
	docRef := fc.client.Collection("threads").Doc(threadID)
	_, err := docRef.Update(ctx, []firestore.Update{
		{Path: "members", Value: firestore.ArrayRemove(userID)},
		{Path: "updatedAt", Value: time.Now()},
	})
	return err
}

// WriteMessage writes a message document to a thread's messages subcollection.
func (fc *FirestoreClient) WriteMessage(ctx context.Context, threadID string, msg map[string]interface{}) (string, error) {
	messagesRef := fc.client.Collection("threads").Doc(threadID).Collection("messages")

	// Auto-generate document ID
	docRef := messagesRef.NewDoc()
	msg["id"] = docRef.ID
	msg["threadId"] = threadID
	msg["isRead"] = false
	msg["createdAt"] = time.Now()
	msg["timestamp"] = time.Now()

	_, err := docRef.Set(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("write message: %w", err)
	}

	// Update thread's last message metadata
	threadRef := fc.client.Collection("threads").Doc(threadID)
	_, _ = threadRef.Update(ctx, []firestore.Update{
		{Path: "lastMessage", Value: msg["content"]},
		{Path: "lastMessageTime", Value: time.Now()},
		{Path: "updatedAt", Value: time.Now()},
	})

	return docRef.ID, nil
}
