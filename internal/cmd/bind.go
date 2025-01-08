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
				Description:  "The channel to bind to",
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

	edit := discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{{
			Title:       "Bind",
			Description: "Failed to bind, please try again",
		}},
	}

	// Parse channelID
	channel := i.ApplicationCommandData().Options[0].ChannelValue(nil)

	// Check bot is able to message in that channel
	perms, err := s.State.UserChannelPermissions(s.State.User.ID, channel.ID)
	if err != nil {
		EditReply(s, i, &edit)
		return
	}
	if perms&discordgo.PermissionSendMessages == 0 {
		(*edit.Embeds)[0].Description =
			"Failed to bind, missing send message permissions for that channel"
		EditReply(s, i, &edit)
		return
	}

	channelID, err := strconv.ParseInt(channel.ID, 10, 64)
	if err != nil {
		EditReplyUnexpected(s, i)
		return
	}

	// Bind and create embed reply
	if err := db.SetChannelID(guildID, channelID); err == nil {
		(*edit.Embeds)[0].Description = "Successfully bound to " + channel.Mention()
	}

	EditReply(s, i, &edit)
}
