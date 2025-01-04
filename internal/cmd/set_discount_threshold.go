package cmd

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/db"
)

// NewSetDiscountThreshold creates /set_discount_threshold <threshold>.
func NewSetDiscountThreshold() Cmd {
	min := float64(1)
	return Cmd{
		Name:        "set_discount_threshold",
		Description: "Set the minimum discount required to trigger a sale alert",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "threshold",
				Description: "The minimum discount required to trigger a sale alert",
				Required:    true,
				MinValue:    &min,
				MaxValue:    99,
			},
		},
		Handle: setDiscountThresholdHandler,
	}
}

func setDiscountThresholdHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	DeferMsgReply(s, i)

	// Parse discount threshold
	threshold := i.ApplicationCommandData().Options[0].IntValue()

	// Parse guildID
	guildID, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		EditReplyUnexpected(s, i)
		return
	}

	// Set threshold and write reply embed
	var description string
	if err = db.SetThreshold(guildID, int(threshold)); err != nil {
		description = "Failed to update discount threshold, please try again"
	} else {
		description = "Successfully updated discount threshold"
	}

	EditReply(s, i, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title:       "Set Discount Threshold",
				Description: description,
			},
		},
	})
}
