package main

import (
	"fmt"

	"github.com/jasonly027/steam_sale_discord_bot_go/internal/steambot"
	"github.com/spf13/viper"
)

func main() {
	viper.AutomaticEnv()
	viper.SetConfigFile(".env")
	viper.ReadInConfig()

	token := viper.GetString("DISCORD_API_KEY")
	if token == "" {
		panic("Discord API token not set as env variable or in .env")
	}

	guild := viper.GetString("DISCORD_DEV_GUILDID")
	if guild != "" {
		fmt.Println("Dev Mode - Registering commands to test guild", guild)
	} else {
		fmt.Println("Prod Mode - Registering commands globally")
	}

	steambot.New(token, guild).Start()
}
