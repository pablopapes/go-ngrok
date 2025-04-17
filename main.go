package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type AuthResponse struct {
	Token string `json:"token"`
}

type NgrokTunnel struct {
	PublicURL string `json:"public_url"`
}

type NgrokResponse struct {
	Tunnels []NgrokTunnel `json:"tunnels"`
}

func goDotEnvVariable(key string) string {

	err := godotenv.Load("./.env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	return os.Getenv(key)
}

func main() {

	api_url := goDotEnvVariable("API_URL")
	username := goDotEnvVariable("USERNAME_API")
	password := goDotEnvVariable("PASSWORD_API")

	authPayload := map[string]string{
		"username": username,
		"password": password,
	}
	authBody, _ := json.Marshal(authPayload)

	authResp, err := http.Post(api_url+"/authentication_token", "application/json", bytes.NewBuffer(authBody))
	if err != nil {
		panic("Error auth: " + err.Error())
	}
	defer authResp.Body.Close()

	bodyBytes, _ := io.ReadAll(authResp.Body)

	if authResp.StatusCode != 200 {
		panic("Error auth: " + string(bodyBytes))
	}

	var auth AuthResponse
	json.Unmarshal(bodyBytes, &auth)

	fmt.Println("Token JWT success")

	ngrokResp, err := http.Get("http://127.0.0.1:4040/api/tunnels")
	if err != nil {
		panic("Error ngrok: " + err.Error())
	}
	defer ngrokResp.Body.Close()

	ngrokBody, _ := io.ReadAll(ngrokResp.Body)
	var tunnels NgrokResponse
	json.Unmarshal(ngrokBody, &tunnels)

	if len(tunnels.Tunnels) == 0 {
		panic("The ngrok tunnel is not running")
	}

	publicURL := tunnels.Tunnels[0].PublicURL
	fmt.Println("Public ngrok url:", publicURL)

	data := map[string]string{
		"value": publicURL + "/",
	}
	dataBody, _ := json.Marshal(data)

	req, _ := http.NewRequest("POST", api_url+"/api/config/url", bytes.NewBuffer(dataBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+auth.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic("Error send URL: " + err.Error())
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println("Response:", string(respBody))
}
