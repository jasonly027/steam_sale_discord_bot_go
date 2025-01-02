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

	fields := []*discordgo.MessageEmbedField{}

	// Add successful apps field
	builder := strings.Builder{}
	for _, app := range succ {
		builder.WriteString(fmt.Sprintf("%s (%d)\n", app.Name, app.Appid))
	}
	if succStr := builder.String(); succStr != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Successfully added",
			Value: succStr,
		})
	}

	// Add unsuccessful apps field
	builder.Reset()
	for _, id := range invalidAppids {
		builder.WriteString(id + "\n")
	}
	for _, app := range fail {
		builder.WriteString(fmt.Sprintf("%d\n", app.Appid))
	}
	if failStr := builder.String(); failStr != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Failed to add",
			Value: failStr,
		})
	}

	EditReply(s, i, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title:  fmt.Sprintf("Added %d out of %d apps", len(succ), len(succ)+len(fail)),
				Fields: fields,
			},
		},
	})
}

// parseAppids iterates through ss and, under the assumption they
// are appid's, tries to construct App's with them.
// Successfully created App's and the failed strings from ss are returned
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
