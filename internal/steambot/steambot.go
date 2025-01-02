// steambot provides a Steam Sale Discord bot.
package steambot

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/cmd"
)

var _ = fmt.Println

type SteamBot struct {
	*discordgo.Session
	gid  string
	cmds map[string]cmd.Cmd
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
		Session: dg,
		gid:     guild,
		cmds:    make(map[string]cmd.Cmd),
	}

	b.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.GuildID == "" {
			cmd.NewReplyHandler(
				&discordgo.InteractionResponseData{
					Content: "Please try commands in a server"},
			)(s, i)
		}

		if cmd, ok := b.cmds[i.ApplicationCommandData().Name]; ok {
			cmd.Handler(s, i)
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
		cmd.NewHelp(),
		cmd.NewAddApps(),
	)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	b.Close()
}

// register registers commands for the bot.
func (b *SteamBot) register(newCmds ...cmd.Cmd) {
	// Add new cmds to map
	for _, newCmd := range newCmds {
		_, exists := b.cmds[newCmd.Name]
		if exists {
			log.Fatal("Command [" + newCmd.Name + "] already exists")
		}

		b.cmds[newCmd.Name] = newCmd
	}

	// Create ApplicationCommands from new cmds
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
