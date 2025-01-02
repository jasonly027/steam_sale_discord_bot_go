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
		log.Fatal("MONGODB_URI not set as env variable or in .env")
	}
	dbName := os.Getenv("MONGODB_DBNAME")
	if dbName == "" {
		log.Fatal("MONGODB_DBNAME not set as env variable or in .env")
	}

	db.Init(uri, dbName)
	defer db.Close()

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("Discord API token not set as env variable or in .env")
	}
	guild := os.Getenv("DISCORD_DEV_GUILDID")
	if guild == "" {
		fmt.Println("Prod Mode - Registering commands globally")
	} else {
		fmt.Println("Dev Mode - Registering commands to test guild", guild)
	}

	steambot.New(token, guild).Start()
}
