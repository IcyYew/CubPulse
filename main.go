package main

import (
	"fmt"
	"net/http"
	"io"
	"errors"
	"encoding/json"
	//"strconv"
	"time"
	"github.com/jackc/pgx/v5"
	"log"
	"os"
	"github.com/joho/godotenv"
	"context"
)

type AppIDResponse map[string]struct {
	Success bool `json:"success"`
	Data struct {
		Name string `json:"name"`
		Appid int `json:"steam_appid"`
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
// Checks if appid a valid Steam appid, returns appname 
func IsValidSteamAppID(appid int) (string, error) {
	url := fmt.Sprintf("http://store.steampowered.com/api/appdetails?appids=%v", appid)
	res, err := steamClient.Get(url)
	if err != nil {
		return "", err
	}
	var appIDResponse AppIDResponse
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		//fmt.Println("Failed to parse response body")
		return "", err
	}
	//fmt.Println("status:", res.StatusCode)
	//fmt.Println("body:", string(body))

	if err := json.Unmarshal(body, &appIDResponse); err != nil {
		//fmt.Println("Failed to unmarshal body")
		return "", err
	}

	// iterate through response entries, looking for an appid match, this is needed for edge case weirdness,
	// like the struct comment where we have response keyed on DLC rather than appid
	for _, value := range appIDResponse {
		if value.Data.Appid == appid {
			if value.Success {
				return value.Data.Name, nil
			} else {
				return "", errors.New("Invalid")
			}
		}

	}
	return "", errors.New("Something went wrong")

}

// hits official web api for a games player count
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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env")
	}
	conn, err := pgx.Connect(context.Background(), os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal("Error loading DB")
	}
	defer conn.Close(context.Background())
	var appname string
	appid := 440
	err = conn.QueryRow(context.Background(), "SELECT app_name FROM steam_apps WHERE appid=$1", appid).Scan(&appname)
	if errors.Is(err, pgx.ErrNoRows) {
		//fmt.Println("App id not found in cache")
		appname, err = IsValidSteamAppID(appid)
		if err != nil {
			log.Fatal("Invalid app id")
		}
		_, err = conn.Exec(
			context.Background(), 
			"INSERT INTO steam_apps (appid, app_name) VALUES ($1, $2)",
			appid,
			appname,
		)
		if err != nil   {
			log.Fatal("Failed to write into steam_apps table")
		}
	} else if err != nil {
		log.Fatal("DB failure, unclassified")
	}

	playercountbody, err := GetCurrentPlayerCount(appid)
	fmt.Printf("%v players currently playing %s\n", playercountbody, appname)
}
