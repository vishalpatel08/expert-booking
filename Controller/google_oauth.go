package controller

import (
	auth "BookingPlatfrom/Auth"
	models "BookingPlatfrom/Models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func googleConfig() *oauth2.Config {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		log.Println("[WARN] Missing Google OAuth env vars: GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET/GOOGLE_REDIRECT_URL")
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

func GoogleLogin(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	if role == "" {
		role = string(models.RoleClient)
	}
	state := url.QueryEscape(role)
	authURL := googleConfig().AuthCodeURL(state, oauth2.AccessTypeOffline)
	fmt.Println("Sending Redirect URI to Google:", googleConfig().RedirectURL)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func GoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		http.Error(w, "Missing code", http.StatusBadRequest)
		return
	}

	cfg := googleConfig()
	tok, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		log.Println("oauth exchange error:", err)
		http.Error(w, "OAuth exchange failed", http.StatusBadRequest)
		return
	}

	client := cfg.Client(context.Background(), tok)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Println("userinfo request error:", err)
		http.Error(w, "Failed to fetch user info", http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	var info struct {
		Email      string `json:"email"`
		Verified   bool   `json:"verified_email"`
		GivenName  string `json:"given_name"`
		FamilyName string `json:"family_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		log.Println("userinfo decode error:", err)
		http.Error(w, "Invalid user info", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(info.Email) == "" {
		http.Error(w, "Email not provided by Google", http.StatusBadRequest)
		return
	}

	firstName := strings.TrimSpace(info.GivenName)
	lastName := strings.TrimSpace(info.FamilyName)
	if firstName == "" && lastName == "" {
		parts := strings.SplitN(info.Email, "@", 2)
		localPart := parts[0]
		localPart = strings.ReplaceAll(localPart, ".", " ")
		firstName = strings.Title(localPart)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err = UserCollection.FindOne(ctx, bson.M{"email": info.Email}).Decode(&user)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			log.Println("mongo find user error:", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		role := models.RoleClient
		if state == string(models.RoleProvider) {
			role = models.RoleProvider
		}
		user = models.User{
			Id:          primitive.NewObjectID(),
			FirstName:   firstName,
			LastName:    lastName,
			Email:       info.Email,
			PhoneNumber: "",
			Role:        role,
			Bookings:    []primitive.ObjectID{},
			CreatedAt:   time.Now().Format("2006-01-02,15:04:05"),
			UpdatedAt:   time.Now().Format("2006-01-02,15:04:05"),
		}
		if _, insErr := UserCollection.InsertOne(ctx, user); insErr != nil {
			log.Println("mongo insert user error:", insErr)
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
	} else {
		updateNeeded := false
		update := bson.M{}
		if strings.TrimSpace(user.FirstName) == "" && firstName != "" {
			update["firstName"] = firstName
			updateNeeded = true
		}
		if strings.TrimSpace(user.LastName) == "" && lastName != "" {
			update["lastName"] = lastName
			updateNeeded = true
		}
		if updateNeeded {
			update["updatedAt"] = time.Now().Format("2006-01-02,15:04:05")
			_, updErr := UserCollection.UpdateOne(ctx, bson.M{"_id": user.Id}, bson.M{"$set": update})
			if updErr != nil {
				log.Println("mongo update user error:", updErr)
			} else {
				if v, ok := update["firstName"].(string); ok {
					user.FirstName = v
				}
				if v, ok := update["lastName"].(string); ok {
					user.LastName = v
				}
				user.UpdatedAt = update["updatedAt"].(string)
			}
		}
	}

	jwtToken, err := auth.JwtGenerate(user.Id, user.Role)
	if err != nil {
		log.Println("jwt generate error:", err)
		http.Error(w, "Failed to issue token", http.StatusInternalServerError)
		return
	}

	frontendRedirect := os.Getenv("FRONTEND_OAUTH_REDIRECT")
	if frontendRedirect == "" {
		frontendRedirect = "http://localhost:5173/oauth/callback"
	}
	redir, parseErr := url.Parse(frontendRedirect)
	if parseErr != nil {
		log.Println("invalid FRONTEND_OAUTH_REDIRECT:", parseErr)
		http.Error(w, "Invalid redirect", http.StatusInternalServerError)
		return
	}
	q := redir.Query()
	q.Set("token", jwtToken)
	q.Set("firstName", user.FirstName)
	q.Set("lastName", user.LastName)
	q.Set("email", user.Email)
	q.Set("role", string(user.Role))
	q.Set("userId", user.Id.Hex())
	redir.RawQuery = q.Encode()
	http.Redirect(w, r, redir.String(), http.StatusFound)
}
