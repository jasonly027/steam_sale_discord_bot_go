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
		Description: "List apps being tracked and their discount thresholds",
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

	// Get apps and create embed reply
	records, err := db.AppsOf(guildID)
	var description string
	var footer *discordgo.MessageEmbedFooter
	switch {
	case err != nil:
		description = "Failed to get apps, please try again"

	case len(records) == 0:
		description = "List is empty! Try adding some apps"

	default:
		sb := strings.Builder{}
		for _, rec := range records {
			sb.WriteString(fmt.Sprintf("%s (%d)", rec.AppName, rec.Appid))
			if rec.AppSaleThreshold > 0 {
				sb.WriteString(fmt.Sprintf(" (%d%%)", rec.AppSaleThreshold))
			}
			sb.WriteString("\n")
		}
		description = sb.String()

		footer = &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("General Discount Threshold: %d%%", records[0].SaleThreshold),
		}
	}

	EditReply(s, i, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title:       "List Apps",
				Description: description,
				Footer:      footer,
			},
		},
	})
}
