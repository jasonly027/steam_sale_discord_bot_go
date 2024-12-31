package steambot

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/cmd"
)

var _ = fmt.Println

type SteamBot struct {
	s    *discordgo.Session
	gid  string
	cmds map[string]*cmd.Cmd
}

func New(token, guild string) (b *SteamBot) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		panic("Invalid bot parameters: " + err.Error())
	}

	b = &SteamBot{
		s:    dg,
		gid:  guild,
		cmds: make(map[string]*cmd.Cmd),
	}

	b.s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if cmd, ok := b.cmds[i.ApplicationCommandData().Name]; ok {
			cmd.Handler(s, i)
		}
	})

	return b
}

func (b *SteamBot) Start() {
	err := b.s.Open()
	if err != nil {
		panic("Failed to open session: " + err.Error())
	}

	b.register(
		cmd.NewHelp(),
	)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	b.s.Close()
}

// Register registers commands for the bot
func (b *SteamBot) register(newCmds ...*cmd.Cmd) {
	// Add new cmds to map
	for _, newCmd := range newCmds {
		_, exists := b.cmds[newCmd.Name]
		if exists {
			panic("Command [" + newCmd.Name + "] already exists")
		}

		b.cmds[newCmd.Name] = newCmd
	}

	// Create ApplicationCommands from new cmds
	appCmds := make([]*discordgo.ApplicationCommand, 0, len(b.cmds))
	for _, cmd := range b.cmds {
		appCmds = append(appCmds, cmd.ApplicationCommand())
	}

	// Write ApplicationCommands to API
	_, err := b.s.ApplicationCommandBulkOverwrite(b.s.State.User.ID, b.gid, appCmds)
	if err != nil {
		panic("Failed to register commands: " + err.Error())
	}
}
