package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/db"
)

// NewListApps creates /list_apps.
func NewListApps() Cmd {
	return Cmd{
		Name:        "list_apps",
		Description: "List all apps being tracked",
		Handle:      listAppsHandler,
	}
}

func listAppsHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	DeferMsgReply(s, i)

	// Parse guildID
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		EditReplyUnexpected(s, i)
		return
	}

	// Get apps
	records, err := db.AppsOf(guildID)

	// Create embed reply
	embed := &discordgo.MessageEmbed{Title: "List Apps"}
	switch {
	case err != nil:
		embed.Description = "Failed to get apps, please try again"

	case len(records) == 0:
		embed.Description = "List is empty! Try adding some apps"

	default:
		builder := strings.Builder{}
		for _, rec := range records {
			builder.WriteString(fmt.Sprintf("%s (%d)\n", *rec.AppName, rec.Appid))
		}

		embed.Description = builder.String()
	}

	EditReply(s, i, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}
