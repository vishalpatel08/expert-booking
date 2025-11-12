package main

import (
	controller "BookingPlatfrom/Controller"
	db "BookingPlatfrom/Db"
	router "BookingPlatfrom/Router"
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
	availabilityCollection := client.Database(dbName).Collection(availabilityCol)
	bookingCollection := client.Database(dbName).Collection(bookingCol)
	providerCollection := client.Database(dbName).Collection(providerCol)
	serviceCollection := client.Database(dbName).Collection(serviceCol)
	userCollection := client.Database(dbName).Collection(userCol)

	controller.Init(availabilityCollection, bookingCollection, providerCollection, serviceCollection, userCollection)

	c := cron.New()
	c.AddFunc("@every 1m", controller.SendReminders)
	c.Start()
	log.Println("Cron job scheduler for reminders has been started.")

	r := router.Router()
	allowedOrigins := handlers.AllowedOrigins([]string{"http://localhost:5173, https://expert-booking-y2x7.onrender.com"})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})

	corsHandler := handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders)(r)

	log.Println("Starting server on :4000...")
	log.Fatal(http.ListenAndServe(":4000", corsHandler))

}
