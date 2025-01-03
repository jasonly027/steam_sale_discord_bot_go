package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/db"
)

func NewRemoveApps() Cmd {
	return Cmd{
		Name:        "remove_apps",
		Description: "Remove apps by their appid from the tracker",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type: discordgo.ApplicationCommandOptionString,
				Name: "appids",
				Description: "Comma separated appids to remove from tracker. " +
					"E.g., 400,440,1868140",
				Required:  true,
				MaxLength: 150,
			},
		},
		Handler: removeAppshandler,
	}
}

func removeAppshandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	DeferReply(s, i)

	strs := strings.Split(i.ApplicationCommandData().Options[0].StringValue(), ",")

	// Parse appids
	appids := []int{}
	invalidAppids := []string{}
	for _, str := range strs {
		str = strings.TrimSpace(str)

		appid, err := strconv.Atoi(str)
		if err != nil || appid <= 0 {
			invalidAppids = append(invalidAppids, str)
			continue
		}

		appids = append(appids, appid)
	}

	// Parse guildID
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		ReplyUnexpected(s, i)
	}

	succ, fail := db.RemoveApps(guildID, appids)

	em := &discordgo.MessageEmbed{Title: "Remove Apps"}

	// Add successfully deleted apps field
	sb := strings.Builder{}
	for _, appid := range succ {
		sb.WriteString(fmt.Sprintf("%d\n", appid))
	}
	if succStr := sb.String(); succStr != "" {
		em.Fields = append(em.Fields, &discordgo.MessageEmbedField{
			Name:  "Successfully deleted",
			Value: succStr,
		})
	}

	sb.Reset()
	for _, appid := range invalidAppids {
		sb.WriteString(appid + "\n")
	}
	for _, appid := range fail {
		sb.WriteString(fmt.Sprintf("%d\n", appid))
	}
	if failStr := sb.String(); failStr != "" {
		em.Fields = append(em.Fields, &discordgo.MessageEmbedField{
			Name:  "Failed to remove",
			Value: failStr,
		})
	}

	EditReply(s, i, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{em},
	})
}
