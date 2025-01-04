package cmd

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/db"
)

var clearAppsCompDelete = "clearAppsCompDelete"
var clearAppsCompCancel = "clearAppsCompCancel"

// NewClearApps create /clear_apps
func NewClearApps() Cmd {
	return Cmd{
		Name:        "clear_apps",
		Description: "Clear apps being tracked",
		Handle:      clearAppsHandler,
		CompHandlers: []ComponentHandler{
			{
				Name:   clearAppsCompDelete,
				Handle: clearAppsCompDeleteHandler,
			},
			{
				Name:   clearAppsCompCancel,
				Handle: clearAppsCompCancelHandler,
			},
		},
	}
}

func clearAppsHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	MsgReply(s, i, &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "Clear Apps",
				Description: "Are you sure?",
			},
		},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Delete",
						Style:    discordgo.DangerButton,
						CustomID: clearAppsCompDelete,
					},
					discordgo.Button{
						Label:    "Cancel",
						Style:    discordgo.SecondaryButton,
						CustomID: clearAppsCompCancel,
					},
				},
			},
		},
	})
}

func clearAppsCompDeleteHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	DeferCompReply(s, i)

	// Parse guildID
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		EditReplyUnexpected(s, i)
		return
	}

	// Try to clear apps and write reply embed
	var description string
	if err := db.ClearApps(guildID); err != nil {
		description = "Failed to clear some apps, please try again"
	} else {
		description = "Successfully cleared apps"
	}

	EditReply(s, i, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title:       "Clear Apps",
				Description: description,
			},
		},
		Components: &[]discordgo.MessageComponent{},
	})
}

func clearAppsCompCancelHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	CompReply(s, i, &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "Clear Apps",
				Description: "Operation cancelled",
			},
		},
		Components: []discordgo.MessageComponent{},
	})
}
