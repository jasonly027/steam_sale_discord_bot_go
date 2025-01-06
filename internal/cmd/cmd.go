// cmd provides slash commands and handlers for interactions.
package cmd

import "github.com/bwmarrin/discordgo"

type Cmd struct {
	Name         string
	Description  string
	Options      []*discordgo.ApplicationCommandOption
	Handle       Handler
	CompHandlers []ComponentHandler
}

type Handler func(*discordgo.Session, *discordgo.InteractionCreate)

type ComponentHandler struct {
	Name   string
	Handle Handler
}

func (c *Cmd) ApplicationCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        c.Name,
		Description: c.Description,
		Options:     c.Options,
	}
}

// NewMsgReplyHandler creates a new reply handler for a msg interaction response.
func NewMsgReplyHandler(data *discordgo.InteractionResponseData) Handler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: data,
		})
	}
}

// MsgReply replies to a msg interaction with data.
func MsgReply(s *discordgo.Session, i *discordgo.InteractionCreate,
	data *discordgo.InteractionResponseData) {
	NewMsgReplyHandler(data)(s, i)
}

// MsgReplyUnexpected replies to a msg interaction with a generic error message.
func MsgReplyUnexpected(s *discordgo.Session, i *discordgo.InteractionCreate) {
	MsgReply(s, i, &discordgo.InteractionResponseData{
		Content: "Error: Something unexpected happened",
	})
}

// CompReply replies to a component interaction with data. Effectively, it
// edits the original message the commponent was attached to.
func CompReply(s *discordgo.Session, i *discordgo.InteractionCreate,
	data *discordgo.InteractionResponseData) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: data,
	})
}

// DeferMsgReply tells a msg interaction that it has been acknowledged, and a
// reply will come at a later time. EditReply() needs to be used for the
// message when the reply can finally be made.
func DeferMsgReply(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
}

// DeferCompReply tells a component interaction that it has been acknowledged, and a
// reply will come at a later time. EditReply() needs to be used for the
// message when the reply can finally be made.
func DeferCompReply(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}

// EditReply edits a msg reply or component reply that has already been sent with data.
func EditReply(s *discordgo.Session, i *discordgo.InteractionCreate, data *discordgo.WebhookEdit) {
	s.InteractionResponseEdit(i.Interaction, data)
}

// EditReply edits a msg reply or component reply that has already been
// sent with a generic error message.
func EditReplyUnexpected(s *discordgo.Session, i *discordgo.InteractionCreate) {
	str := "Error: Something unexpected happened"
	EditReply(s, i, &discordgo.WebhookEdit{
		Content: &str,
	})
}
