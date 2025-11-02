package controller

import (
    models "BookingPlatfrom/Models"
    "context"
    "encoding/json"
    "log"
    "net/http"
    "time"

    "github.com/gorilla/mux"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

// GetServicesByProvider returns all services for a given provider id (provider's user id)
func GetServicesByProvider(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    vars := mux.Vars(r)
    idStr, ok := vars["providerId"]
    if !ok || idStr == "" {
        http.Error(w, "providerId is required", http.StatusBadRequest)
        return
    }

    providerObjID, err := primitive.ObjectIDFromHex(idStr)
    if err != nil {
        http.Error(w, "invalid providerId", http.StatusBadRequest)
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    filter := bson.M{"providerId": providerObjID}
    cursor, err := ServiceCollection.Find(ctx, filter)
    if err != nil {
        log.Println("Error querying services:", err)
        http.Error(w, "Failed to fetch services", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(ctx)

    var services []models.Service
    if err := cursor.All(ctx, &services); err != nil {
        log.Println("Error decoding services:", err)
        http.Error(w, "Failed to decode services", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]interface{}{
        "services": services,
    })
}
