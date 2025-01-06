package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/db"
)

// NewRemoveApps creates /remove_apps <appid>,<appid>,...
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
		Handle: removeAppshandler,
	}
}

func removeAppshandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	DeferMsgReply(s, i)

	// Parse appids
	strs := strings.Split(i.ApplicationCommandData().Options[0].StringValue(), ",")
	succ, invalidAppids := strsToAppids(strs)

	// Parse guildID
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		EditReplyUnexpected(s, i)
		return
	}

	// Remove apps and create embed reply
	succ, fail := db.RemoveApps(guildID, succ)
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

	// Add unsuccessfully deleted apps field
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

func strsToAppids(ss []string) (succ []int, fail []string) {
	for _, s := range ss {
		s = strings.TrimSpace(s)

		// Check convertible to int
		appid, err := strconv.Atoi(s)
		if err != nil || appid <= 0 {
			fail = append(fail, s)
			continue
		}

		succ = append(succ, appid)
	}

	return succ, fail
}
