// steambot provides a Steam Sale Discord bot.
package steambot

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/cmd"
)

type SteamBot struct {
	*discordgo.Session
	gid          string
	cmds         map[string]cmd.Cmd
	compHandlers map[string]cmd.Handler
}

// New creates a new Steam bot with a given Discord API bot token.
// An optional Guild ID can be supplied to exclusively register commands to.
// Otherwise, "" can be used to register the commands globally.
// Use b.Start() to start the bot.
func New(token, guild string) (b *SteamBot) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Invalid bot parameters:", err)
	}

	b = &SteamBot{
		Session:      dg,
		gid:          guild,
		cmds:         map[string]cmd.Cmd{},
		compHandlers: map[string]cmd.Handler{},
	}

	b.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			// Bot only works in guilds
			if i.GuildID == "" {
				cmd.MsgReply(s, i,
					&discordgo.InteractionResponseData{
						Content: "Please try commands in a server",
					})
			}

			// Map to command's handler
			if cmd, ok := b.cmds[i.ApplicationCommandData().Name]; ok {
				cmd.Handle(s, i)
			}
		case discordgo.InteractionMessageComponent:
			// Map to component's handler
			if handle, ok := b.compHandlers[i.MessageComponentData().CustomID]; ok {
				handle(s, i)
			}
		}
	})

	return b
}

// Start opens the bot and registers its commands.
func (b *SteamBot) Start() {
	err := b.Open()
	if err != nil {
		log.Fatal("Failed to open session:", err)
	}

	b.register(
		cmd.NewSearch(),
	)

	// Listen for exit
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	b.Close()
}

// register registers commands for the bot.
func (b *SteamBot) register(newCmds ...cmd.Cmd) {
	// Map cmds and their component handlers
	for _, newCmd := range newCmds {
		_, exists := b.cmds[newCmd.Name]
		if exists {
			log.Fatal("Command [" + newCmd.Name + "] already exists")
		}
		b.cmds[newCmd.Name] = newCmd

		for _, handler := range newCmd.CompHandlers {
			_, exists := b.compHandlers[handler.Name]
			if exists {
				log.Fatal("Command handler [" + handler.Name + "] already exists")
			}
			b.compHandlers[handler.Name] = handler.Handle
		}
	}

	// Create ApplicationCommands from cmds
	appCmds := make([]*discordgo.ApplicationCommand, 0, len(b.cmds))
	for _, cmd := range b.cmds {
		appCmds = append(appCmds, cmd.ApplicationCommand())
	}

	// Write ApplicationCommands to API
	_, err := b.ApplicationCommandBulkOverwrite(b.State.User.ID, b.gid, appCmds)
	if err != nil {
		log.Fatal("Failed to register commands: ", err)
	}
}
