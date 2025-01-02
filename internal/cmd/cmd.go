// cmd provides slash commands and handlers for interactions.
package cmd

import "github.com/bwmarrin/discordgo"

type handler func(*discordgo.Session, *discordgo.InteractionCreate)

type Cmd struct {
	Name        string
	Description string
	Options     []*discordgo.ApplicationCommandOption
	Handler     handler
}

func (c *Cmd) ApplicationCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        c.Name,
		Description: c.Description,
		Options:     c.Options,
	}
}

// NewReplyHandler creates a new reply handler for an interaction response.
func NewReplyHandler(data *discordgo.InteractionResponseData) handler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: data,
		})
	}
}

// ReplyUnexpected replies to an interaction with a generic error message.
func ReplyUnexpected(s *discordgo.Session, i *discordgo.InteractionCreate) {
	NewReplyHandler(&discordgo.InteractionResponseData{
		Content: "Error: Something unexpected happened",
	})(s, i)
}

// DeferReply tells an interaction that it has been acknowledged, and a
// reply will come at a later time. EditReply() needs to be used for the
// message when a reply can finally be made.
func DeferReply(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
}

// EditReply edits a reply that has already been sent.
func EditReply(s *discordgo.Session, i *discordgo.InteractionCreate, data *discordgo.WebhookEdit) {
	s.InteractionResponseEdit(i.Interaction, data)
}
