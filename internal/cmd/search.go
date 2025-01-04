package cmd

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/db"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/steam"
)

var searchCompConfirm = "searchCompConfirm"
var searchCompCancelStr = "--- Cancel Adding App ---"

// NewSearch creates /search <query>.
func NewSearch() Cmd {
	return Cmd{
		Name:        "search",
		Description: "Search for an app to add to the tracker",
		Handle:      searchHandler,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "query",
				Description: "Search query used to find an app",
				Required:    true,
				MaxLength:   100,
			},
		},
		CompHandlers: []ComponentHandler{
			{
				Name:   searchCompConfirm,
				Handle: searchCompConfirmHandler,
			},
		},
	}
}

func searchHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	DeferMsgReply(s, i)

	// Search Steam apps with query
	query := i.ApplicationCommandData().Options[0].StringValue()
	res, err := steam.Search(query)
	if err != nil {
		EditReply(s, i, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				{
					Title:       "Search",
					Description: "Failed to get search results, please try again later",
				},
			},
		})
		return
	}

	// Create search results select menu
	options := make([]discordgo.SelectMenuOption, 0, len(res))
	for _, r := range res {
		options = append(options, discordgo.SelectMenuOption{
			Label: fmt.Sprintf("%s (%d)", r.Name, r.Appid),
			Value: fmt.Sprint(r.Appid),
		})
	}
	options = append(options, discordgo.SelectMenuOption{
		Label: searchCompCancelStr,
		Value: searchCompCancelStr,
	})

	EditReply(s, i, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title:       "Search",
				Description: "Select an app below",
			},
		},
		Components: &[]discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						MenuType: discordgo.StringSelectMenu,
						CustomID: searchCompConfirm,
						Options:  options,
					},
				},
			},
		},
	})
}

func searchCompConfirmHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	choice := i.MessageComponentData().Values[0]
	if choice == clearAppsCompCancel {
		CompReply(s, i, &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Search",
					Description: "Cancelled adding app",
				},
			},
			Components: []discordgo.MessageComponent{},
		})
		return
	}

	DeferCompReply(s, i)

	edit := discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title:       "Search",
				Description: "Error adding app, please try again",
			},
		},
		Components: &[]discordgo.MessageComponent{},
	}

	succ, _ := parseAppids([]string{choice})
	if len(succ) != 1 {
		EditReply(s, i, &edit)
		return
	}

	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		EditReplyUnexpected(s, i)
		return
	}

	succ, _ = db.AddApps(guildID, succ)
	if len(succ) != 1 {
		EditReply(s, i, &edit)
		return
	}

	(*edit.Embeds)[0].Description = "Successfully added app"
	EditReply(s, i, &edit)
}
