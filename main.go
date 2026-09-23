package main

import (
	"fmt"
	"net/http"
	"io"
	"errors"
	"encoding/json"
	"github.com/joho/godotenv"
	"log"
)

type AppIDResponse struct {
	Success string `json:"success"`
}

func IsValidSteamAppID(appid int) (int, error) {
	url := fmt.Sprintf("http://store.steampowered.com/api/appdetails?appids=%v", appid)
	res, err := http.Get(url)
	if err != nil {
		return -1, err
	}
	var success AppIDResponse
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return -1, err
	}

	if err := json.Unmarshal(body, &success); err != nil {
		return -1, err
	}
	if success.Success == "false" {
		return -1, errors.New("Not valid steam app id")
	}
	return appid, nil
}

func GetCurrentPlayerCount(appid int) (string, error) {
	url := fmt.Sprintf("https://api.steampowered.com/ISteamUserStats/GetNumberOfCurrentPlayers/v1/?appid=%v", appid)
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}


func main() {

	appid, err := IsValidSteamAppID(440)
	if err != nil {
		fmt.Printf("Invalid Steam app id: %v", appid)
	}
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Err loading .env file")
	}

	fmt.Printf("Valid steam app id: %v", appid)

	playercountbody, err := GetCurrentPlayerCount(appid)
	fmt.Printf("%s", playercountbody)
}
