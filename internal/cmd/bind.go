package cmd

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/db"
)

func NewBind() Cmd {
	return Cmd{
		Name:        "bind",
		Description: "Set the channel where alerts are sent",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionChannel,
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
				Name:         "channel",
				Required:     true,
			},
		},
		Handle: bindHandler,
	}
}

func bindHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	DeferMsgReply(s, i)

	// Parse guildID
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		EditReplyUnexpected(s, i)
		return
	}

	// Parse channelID
	channel := i.ApplicationCommandData().Options[0].ChannelValue(nil)
	channelID, err := strconv.ParseInt(channel.ID, 10, 64)
	if err != nil {
		EditReplyUnexpected(s, i)
		return
	}

	// Bind and create embed reply
	err = db.SetChannelID(guildID, channelID)
	var embed *discordgo.MessageEmbed
	if err != nil {
		embed = &discordgo.MessageEmbed{
			Title:       "Bind",
			Description: "Failed to bind, please try again",
		}
	} else {
		embed = &discordgo.MessageEmbed{
			Title:       "Bind",
			Description: "Successfully bound to " + channel.Mention(),
		}
	}

	EditReply(s, i, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}})
}
