package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

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

type publicUrl struct {
	Url string `json:"url"`
}

func NewConfig() (publicUrl, error) {
	var cf publicUrl
	confFile, err := os.Open("./config.json")

	if err != nil {
		if os.IsNotExist(err) {
			defaultConfig := publicUrl{
				Url: "",
			}
			configFile, err := os.Create("./config.json")
			if err != nil {
				return cf, err
			}
			defer configFile.Close()

			encoder := json.NewEncoder(configFile)
			if err := encoder.Encode(&defaultConfig); err != nil {
				return cf, err
			}

			// Reopen the newly created file
			confFile, err = os.Open("./config.json")
			if err != nil {
				return cf, err
			}
		} else {
			log.Fatal(err)
		}
	}

	defer confFile.Close()

	jsonParser := json.NewDecoder(confFile)
	if err := jsonParser.Decode(&cf); err != nil {
		return cf, err
	}

	return cf, nil
}

func updateConfigFile(urlAddress string) {
	cfg, err := NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	cfg.Url = urlAddress

	file, err := json.MarshalIndent(cfg, "", " ")

	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile("./config.json", file, 0644)

	if err != nil {
		log.Fatal(err)
	}
}

func runOnce() {

	api_url := os.Getenv("API_URL")
	username := os.Getenv("USERNAME_API")
	password := os.Getenv("PASSWORD_API")

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

	// Check if the URL is the same as in the config file
	var cfg, errcf = NewConfig()
	if errcf != nil {
		log.Fatal(errcf)
	}

	if cfg.Url == publicURL {
		log.Println("The url is the same. Exiting...")
		return
	}

	updateConfigFile(publicURL)

	// send Url to API
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

func main() {
	if err := godotenv.Load("./.env"); err != nil {
		log.Fatal("can't load .env")
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	runOnce()

	// Loop
	for range ticker.C {
		runOnce()
	}
}
