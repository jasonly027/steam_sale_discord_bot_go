package cmd

import "github.com/bwmarrin/discordgo"

type Cmd struct {
	Name        string
	Description string
	Handler     func(*discordgo.Session, *discordgo.InteractionCreate)
}

func (c *Cmd) ApplicationCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        c.Name,
		Description: c.Description,
	}
}
