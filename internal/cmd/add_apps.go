package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/db"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/steam"
)

// NewAddApps creates /add_apps <appid>,<appid>,...
func NewAddApps() Cmd {
	return Cmd{
		Name:        "add_apps",
		Description: "Add apps by their appid to the tracker",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type: discordgo.ApplicationCommandOptionString,
				Name: "appids",
				Description: "Comma separated appids to add to tracker. " +
					"E.g., 400,440,1868140",
				Required:  true,
				MaxLength: 150,
			},
		},
		Handler: addAppsHandler,
	}
}

// addAppsHandler is the handler for the add_apps command
func addAppsHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	DeferReply(s, i)

	// Parse appids
	strs := strings.Split(i.ApplicationCommandData().Options[0].StringValue(), ",")
	apps, invalidAppids := parseAppids(strs)

	// Add apps to database
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		ReplyUnexpected(s, i)
	}
	succ, fail := db.AddApps(guildID, apps)

	em := &discordgo.MessageEmbed{Title: "Add Apps"}

	// Add successful apps field
	sb := strings.Builder{}
	for _, app := range succ {
		sb.WriteString(fmt.Sprintf("%s (%d)\n", app.Name, app.Appid))
	}
	if succStr := sb.String(); succStr != "" {
		em.Fields = append(em.Fields, &discordgo.MessageEmbedField{
			Name:  "Successfully added",
			Value: succStr,
		})
	}

	// Add unsuccessful apps field
	sb.Reset()
	for _, id := range invalidAppids {
		sb.WriteString(id + "\n")
	}
	for _, app := range fail {
		sb.WriteString(fmt.Sprintf("%d\n", app.Appid))
	}
	if failStr := sb.String(); failStr != "" {
		em.Fields = append(em.Fields, &discordgo.MessageEmbedField{
			Name:  "Failed to add",
			Value: failStr,
		})
		em.Footer = &discordgo.MessageEmbedFooter{
			Text: "Note: Make sure failed appids are either priced or are yet to be released",
		}
	}

	EditReply(s, i, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{em},
	})
}

// parseAppids iterates through ss and, tries to create
// valid App's with them. Valid apps need to have the price_overview
// field set or haven't been released yet
func parseAppids(ss []string) (succ []steam.App, fail []string) {
	succ = []steam.App{}
	fail = []string{}

	for _, s := range ss {
		s = strings.TrimSpace(s)

		appid, err := strconv.Atoi(s)
		if err != nil || appid < 0 {
			fail = append(fail, s)
			continue
		}

		app, err := steam.NewApp(appid)
		if err != nil || (app.Initial == "" && !app.ComingSoon) {
			fail = append(fail, s)
			continue
		}

		succ = append(succ, app)
	}

	return succ, fail
}
