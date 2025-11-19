package main

import (
	controller "BookingPlatfrom/Controller"
	db "BookingPlatfrom/Db"
	router "BookingPlatfrom/Router"
	"BookingPlatfrom/websocket"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

func main() {
	fmt.Println("Welcome to Expert-Booking")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	client, err := db.ConnectDB()
	if err != nil {
		log.Fatal("Error connecting to mongo DB", err)
	}

	dbName := os.Getenv("DB_NAME")
	availabilityCol := os.Getenv("AVAILABILITY_COL")
	bookingCol := os.Getenv("BOOKING_COL")
	providerCol := os.Getenv("PROVIDER_COL")
	serviceCol := os.Getenv("SERVICE_COL")
	userCol := os.Getenv("USER_COL")
	chatCol := "chats"

	// Initialize collections
	availabilityCollection := client.Database(dbName).Collection(availabilityCol)
	bookingCollection := client.Database(dbName).Collection(bookingCol)
	providerCollection := client.Database(dbName).Collection(providerCol)
	serviceCollection := client.Database(dbName).Collection(serviceCol)
	userCollection := client.Database(dbName).Collection(userCol)
	chatCollection := client.Database(dbName).Collection(chatCol)

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Initialize controllers
	controller.Init(availabilityCollection, bookingCollection, providerCollection, serviceCollection, userCollection)
	chatController := controller.NewChatController(chatCollection)

	// Initialize router with hub and chat controller
	r := router.NewRouter(hub, chatController)

	// Start cron job for reminders
	c := cron.New()
	c.AddFunc("@every 1m", controller.SendReminders)
	c.Start()
	log.Println("Cron job scheduler for reminders has been started.")

	// All routes including WebSocket are now handled by the router
	http.Handle("/", r)

	// CORS configuration
	allowedOrigins := handlers.AllowedOrigins([]string{"http://localhost:5173", "https://expert-booking-y2x7.onrender.com"})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})

	handler := handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders)(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}

	log.Printf("Server started on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
