package main

import (
	"fmt"
	"net/http"
	"io"
	"errors"
	"encoding/json"
	"strconv"
	"time"
)

type AppIDResponse map[string]struct {
	Success bool `json:"success"`
	Data struct {
		Name string `json:"name"`
	}
}

type PlayerCountResponse struct {
	Response struct {
		Playercount int `json:"player_count"`
		Result int `json:"result"`
	} `json:"response"`
}

var steamClient = &http.Client{
	Timeout: 5 * time.Second,
}
// Checks if appid a valid Steam appid, returns appid and app name
func IsValidSteamAppID(appid int) (int, string, error) {
	url := fmt.Sprintf("http://store.steampowered.com/api/appdetails?appids=%v", appid)
	res, err := steamClient.Get(url)
	if err != nil {
		return -1, "", err
	}
	var appIDResponse AppIDResponse
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return -1, "", err
	}

	if err := json.Unmarshal(body, &appIDResponse); err != nil {
		return -1, "", err
	}

	if value, exists := appIDResponse[strconv.Itoa(appid)]; exists {
		if value.Success {
			return appid, value.Data.Name, nil
		} else {
			return -1, "", errors.New("AppID exists, but not valid ID")
		}
	} else {
		return -1, "", errors.New("Bad Steam response for AppID")
	}
}

func GetCurrentPlayerCount(appid int) (int, error) {
	url := fmt.Sprintf("https://api.steampowered.com/ISteamUserStats/GetNumberOfCurrentPlayers/v1/?appid=%v", appid)
	res, err := steamClient.Get(url)
	if err != nil {
		return -1, err
	}
	var playercount PlayerCountResponse
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return -1, err
	}
	if err := json.Unmarshal(body, &playercount); err != nil {
		return -1, err
	}

	if playercount.Response.Result > 0 {
		return playercount.Response.Playercount, nil
	} else {
		return -1, errors.New("Steam returned bad result for AppID")
	}
}


func main() {

	appid, appname, err := IsValidSteamAppID(440)
	if err != nil {
		fmt.Printf("Invalid Steam app id: %v", appid)
	}

	//fmt.Printf("Valid steam app id: %v\n", appid)

	playercountbody, err := GetCurrentPlayerCount(appid)
	fmt.Printf("%v players currently playing %s\n", playercountbody, appname)
}
