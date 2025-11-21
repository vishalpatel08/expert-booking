package controller

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ChatMessage struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	SenderID   string             `bson:"senderId" json:"senderId"`
	ReceiverID string             `bson:"receiverId" json:"receiverId"`
	Content    string             `bson:"content" json:"content"`
	Timestamp  time.Time          `bson:"timestamp" json:"timestamp"`
}

type ChatController struct {
	chatCollection *mongo.Collection
}

func NewChatController(chatCollection *mongo.Collection) *ChatController {
	return &ChatController{
		chatCollection: chatCollection,
	}
}

// GetChatHistory returns the chat history between two users
func (cc *ChatController) GetChatHistory(w http.ResponseWriter, r *http.Request) {
	senderID := r.URL.Query().Get("senderId")
	receiverID := r.URL.Query().Get("receiverId")

	if senderID == "" || receiverID == "" {
		http.Error(w, "senderId and receiverId are required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"$or": []bson.M{
			{
				"senderId":   senderID,
				"receiverId": receiverID,
			},
			{
				"senderId":   receiverID,
				"receiverId": senderID,
			},
		},
	}

	cursor, err := cc.chatCollection.Find(ctx, filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var messages []ChatMessage
	if err = cursor.All(ctx, &messages); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (cc *ChatController) SaveMessage(message ChatMessage) error {
	message.Timestamp = time.Now()
	_, err := cc.chatCollection.InsertOne(context.Background(), message)
	return err
}

func (cc *ChatController) GetRecentChats(userID string) ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pipeline := mongo.Pipeline{
		bson.D{{
			Key: "$match",
			Value: bson.D{{
				Key: "$or",
				Value: bson.A{
					bson.D{{Key: "senderId", Value: userID}},
					bson.D{{Key: "receiverId", Value: userID}},
				},
			}},
		}},
		bson.D{{
			Key:   "$sort",
			Value: bson.D{{Key: "timestamp", Value: -1}},
		}},
	}

	cursor, err := cc.chatCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var recentChats []map[string]interface{}
	seen := make(map[string]bool)

	for cursor.Next(ctx) {
		var result map[string]interface{}
		if err := cursor.Decode(&result); err != nil {
			log.Printf("Error decoding chat: %v", err)
			continue
		}

		var otherUserID string
		if result["senderId"] == userID {
			otherUserID = result["receiverId"].(string)
		} else {
			otherUserID = result["senderId"].(string)
		}

		if !seen[otherUserID] {
			recentChats = append(recentChats, map[string]interface{}{
				"userId":      otherUserID,
				"lastMessage": result["content"],
				"timestamp":   result["timestamp"],
			})
			seen[otherUserID] = true
		}

		if len(recentChats) >= 20 {
			break
		}
	}

	return recentChats, nil
}
