package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jasonly027/steam_sale_discord_bot_go/internal/db"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/steambot"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		if data, err := os.ReadFile(os.Getenv("MONGODB_URI_FILE")); err == nil {
			uri = string(data)
		} else {
			log.Fatal("MONGODB_URI not set as env variable or in .env, and MONGODB_URI_FILE not readable")
		}
	}

	dbName := os.Getenv("MONGODB_DBNAME")
	if dbName == "" {
		if data, err := os.ReadFile(os.Getenv("MONGODB_DBNAME_FILE")); err == nil {
			dbName = string(data)
		} else {
			log.Fatal("MONGODB_DBNAME not set as env variable or in .env, and MONGODB_DBNAME_FILE not readable")
		}
	}

	db.Init(uri, dbName)
	defer db.Close()

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		if data, err := os.ReadFile(os.Getenv("DISCORD_TOKEN_FILE")); err == nil {
			token = string(data)
		} else {
			log.Fatal("Discord API token not set as env variable or in .env, and DISCORD_TOKEN_FILE not readable")
		}
	}
	guild := os.Getenv("DISCORD_DEV_GUILDID")
	if guild == "" {
		fmt.Println("Prod Mode - Registering commands globally")
	} else {
		fmt.Println("Dev Mode - Registering commands to test guild", guild)
	}

	steambot.New(token, guild).Start()
}
