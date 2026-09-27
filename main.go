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
	"os/signal"
	"syscall"
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

type Sample struct {
	Appid int `db:"appid"`
	SampledAt time.Time `db:"sampled_at"`
	PlayerCount int `db:"player_count"`
}


var steamClient = &http.Client{
	Timeout: 5 * time.Second,
}
// Checks if appid a valid Steam appid, returns appname 
func IsValidSteamAppID(ctx context.Context, appid int) (string, error) {
	url := fmt.Sprintf("http://store.steampowered.com/api/appdetails?appids=%v", appid)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req = req.WithContext(ctx)
	res, err := steamClient.Do(req)
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
func GetCurrentPlayerCount(ctx context.Context, appid int) (int, error) {
	url := fmt.Sprintf("https://api.steampowered.com/ISteamUserStats/GetNumberOfCurrentPlayers/v1/?appid=%v", appid)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return -1, err
	}
	req = req.WithContext(ctx)
	res, err := steamClient.Do(req)
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

func run(ctx context.Context) error {
	err := godotenv.Load()
	if err != nil {
		return err
	}
	conn, err := pgx.Connect(ctx, os.Getenv("DB_URL"))
	if err != nil {
		return err
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
		defer cancel()
		conn.Close(cleanupCtx)
	}()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	var appname string
	var samples []Sample
	appid := 440
	err = conn.QueryRow(ctx, "Select app_name FROM steam_apps WHERE appid=$1", appid).Scan(&appname)
	if errors.Is(err, pgx.ErrNoRows) {
		appname, err = IsValidSteamAppID(ctx, appid)
		if err != nil {
			return err
		}
		_, err = conn.Exec(
			ctx,
			"INSERT INTO steam_apps (appid, app_name) VALUES ($1, $2)",
			appid,
			appname,
		)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	failureCounter := 1
	for {
		select {
		case <-ticker.C:
			playercountbody, err := GetCurrentPlayerCount(ctx, appid)
			if err != nil {
				log.Printf("Player count fetch failure %d", failureCounter)
				failureCounter++
				continue
			}
			_, err = conn.Exec(
				ctx,
				"INSERT INTO app_samples (appid, player_count) VALUES ($1, $2)",
				appid,
				playercountbody,
			)
			if err != nil {
				return err
			}
			fmt.Println("Sample inserted.")
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
			defer cancel()
			rows, err := conn.Query(shutdownCtx, "SELECT appid, sampled_at, player_count FROM app_samples WHERE appid = $1 ORDER BY sampled_at", appid)
			if err != nil {
				return err
			} 
			defer rows.Close()

			for rows.Next() {
				var sample Sample
				err = rows.Scan(
					&sample.Appid, 
					&sample.SampledAt, 
					&sample.PlayerCount,
				)
				if err != nil {
					return err
				}
				samples = append(samples, sample)
			}
			if err := rows.Err(); err != nil {
				return err
			}
			fmt.Println("Samples: ")

			for i := 0; i < len(samples); i++ {
				fmt.Println(samples[i])
			}
			return nil 
		}
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		log.Printf("%v", err)
	}
}
