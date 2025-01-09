package cmd

import (
	"github.com/bwmarrin/discordgo"
)

// NewHelp creates the /help command.
func NewHelp() Cmd {
	return Cmd{
		Name:        "help",
		Description: "Show a list of commands and their descriptions",
		Handle: NewMsgReplyHandler(&discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title: "Commands and FAQ",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name: "/bind <text_channel>",
							Value: "Set the channel to where alerts are sent. " +
								"By default, sends to the default channel.",
						},
						{
							Name: "/set_discount_threshold <percentage>",
							Value: "Set the minimum discount percentage warranting an alert of an app sale. " +
								"By default, the threshold is 1%.",
						},
						{
							Name:  "/add_apps <appId,appId,...>",
							Value: "Add comma separated app IDs to the tracker.",
						},
						{
							Name:  "/remove_apps <appId,appId,...>",
							Value: "Remove comma separated app IDs from the tracker.",
						},
						{
							Name:  "/search <query>",
							Value: "Search for an app to add to the tracker.",
						},
						{
							Name:  "/list_apps",
							Value: "List all the apps currently being tracked."},
						{
							Name:  "/clear_apps",
							Value: "Clear the tracking list.",
						},
						{
							Name:   "How often does the bot check for sales?",
							Value:  "The bot checks for sales every day at about **10:05 AM (PDT)**.",
							Inline: true,
						},
						{
							Name: "Why aren't alerts showing up?",
							Value: "Reconfigure your discount threshold in case it is too high. " +
								"Additionally, try rebinding to a text channel.",
							Inline: true,
						},
						{
							Name: "The app is still on sale but there wasn't an alert.",
							Value: "Alerts for an app are only sent on the first day of a sale duration " +
								"or, when added *during* a sale, on the following daily check.",
							Inline: true,
						},
					},
				},
			},
		}),
	}
}
