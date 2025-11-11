package router

import (
	controller "BookingPlatfrom/Controller"

	"github.com/gorilla/mux"
)

func Router() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/registration", controller.UserRegistration).Methods("POST")
	router.HandleFunc("/login", controller.UserLogin).Methods("POST")
	router.HandleFunc("/provider", controller.ProviderRegistration).Methods("POST")
	router.HandleFunc("/providers", controller.GetProviders).Methods("GET")
	router.HandleFunc("/providers/{providerId}/services", controller.GetServicesByProvider).Methods("GET")
	router.HandleFunc("/providers/{providerId}/schedule", controller.GetProviderSchedule).Methods("GET")
	router.HandleFunc("/service", controller.ServiceRegistration).Methods("POST")
	router.HandleFunc("/booking", controller.ServiceBooking).Methods("POST")
	router.HandleFunc("/bookings/me", controller.GetMyBookings).Methods("GET")
	router.HandleFunc("/updateschedule", controller.UpdateProviderSchedule).Methods("PUT")
	router.HandleFunc("/webhooks/notifications", controller.HandleNotificationWebhook).Methods("POST")
	return router
}
