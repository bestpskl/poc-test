package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"server/model"
	"time"

	"github.com/go-oauth2/oauth2/manage"
	"github.com/go-oauth2/oauth2/server"
	"github.com/go-redis/redis"
	"github.com/golang-jwt/jwt"
	"github.com/skip2/go-qrcode"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {

	// set server
	manager := manage.NewDefaultManager()
	srv := server.NewServer(server.NewConfig(), manager)
	_ = srv

	// srv.SetPasswordAuthorizationHandler(func(ctx context.Context, clientID, username, password string) (userID string, err error) {
	// 	if username == "test" && password == "test" {
	// 		userID = "test"
	// 	}
	// 	return
	// }) // อันนี้งง

	// create connection postgres
	dsn := "host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable TimeZone=Asia/Bangkok"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database")
	}

	err = db.AutoMigrate(&model.User{})
	if err != nil {
		fmt.Println("Failed to migrate User model:", err)
		return
	}

	// create connection redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis server address
		Password: "",               // Redis password, if any
		DB:       0,                // Redis database number
	})
	_ = redisClient

	userqr := generateRandomString(10)
	passwordqr := generateRandomString(10)
	user := model.User{}

	// ----- route to CREATE -----

	http.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
		// generate user password
		text := userqr + "|" + passwordqr // must be signed ?

		// generate qr as string
		qrCodeString, err := generateQRCodeString(text)
		if err != nil {
			fmt.Println("Error generating QR code:", err)
			return
		}
		fmt.Println(qrCodeString)
		// give this qrCodeString to frontend to generate picture --->

		// save in db
		user = model.User{
			Username: userqr,
			Password: passwordqr,
			// may be set expired time ?
		}

		err = db.Create(&user).Error
		if err != nil {
			fmt.Println("Failed to create user::", err)
			return
		}
	})

	// ----- route to VALIDATE -----

	http.HandleFunc("/validate", func(w http.ResponseWriter, r *http.Request) {
		// validate
		result := db.Where("username = ? AND password = ?", userqr, passwordqr).First(&user)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				// User not found
				fmt.Println("User not found")
				return
			} else {
				// Other error occurred
				fmt.Println("Error occurred:", result.Error)
				return
			}
		} else {
			// User found

			// generate access token
			secretKey := []byte(userqr + "|" + passwordqr)
			expirationTime := time.Now().Add(1 * time.Hour)

			claims := jwt.StandardClaims{
				ExpiresAt: expirationTime.Unix(),
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			signedToken, err := token.SignedString(secretKey)
			if err != nil {
				fmt.Println("Failed to sign the token:", err)
				return
			}
			fmt.Println("Generated access token:", signedToken)
			// return signedToken (accessToken)

			// save access token on redis
			// ctx := context.Background()
			// err = redisClient.Set(ctx, "Key", signedToken).Err() // change key to identify
			// if err != nil {
			// 	fmt.Println("Error:", err)
			// 	return
			// }
		}
	})
}

// function generate random string
func generateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())

	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	chars := []byte(charset)

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}

	return string(result)
}

// function generate QR code as string
func generateQRCodeString(data string) (string, error) {
	qrCode, err := qrcode.Encode(data, qrcode.Medium, -1)
	if err != nil {
		return "", err
	}

	return string(qrCode), nil
}
