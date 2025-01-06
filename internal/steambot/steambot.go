// steambot provides a Steam Sale Discord bot.
package steambot

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/cmd"
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/db"
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

	b.registerHandlers([]interface{}{
		b.commandHandler,
		b.guildCreateHandler,
		b.guildDeleteHandler,
		b.readyHandler,
	})

	return b
}

// Start opens the bot and registers its commands.
func (b *SteamBot) Start() {
	err := b.Open()
	if err != nil {
		log.Fatal("Failed to open session:", err)
	}

	b.registerCommands([]cmd.Cmd{
		cmd.NewSearch(),
	})

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	b.Close()
}

// registerCommands registers commands for the bot.
func (b *SteamBot) registerCommands(cmds []cmd.Cmd) {
	// Map cmds and their component handlers
	for _, cmd := range cmds {
		_, exists := b.cmds[cmd.Name]
		if exists {
			log.Fatal("Command [" + cmd.Name + "] already exists")
		}
		b.cmds[cmd.Name] = cmd

		for _, handler := range cmd.CompHandlers {
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

func (b *SteamBot) registerHandlers(handlers []interface{}) {
	for _, h := range handlers {
		b.AddHandler(h)
	}
}

func (b *SteamBot) commandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		// Bot only works in guilds
		if i.GuildID == "" {
			cmd.MsgReply(s, i,
				&discordgo.InteractionResponseData{
					Content: "Please try commands in a server",
				})
		}

		if cmd, ok := b.cmds[i.ApplicationCommandData().Name]; ok {
			cmd.Handle(s, i)
		}
	case discordgo.InteractionMessageComponent:
		if handle, ok := b.compHandlers[i.MessageComponentData().CustomID]; ok {
			handle(s, i)
		}
	}
}

func (b *SteamBot) guildCreateHandler(s *discordgo.Session, g *discordgo.GuildCreate) {
	guildID, err := strconv.ParseInt(g.ID, 10, 64)
	if err != nil {
		return
	}

	channelID, err := defaultTextChannel(s, g.Channels)
	if err != nil {
		// pass channelID of 0 anyways (will be unset in the record)
	}

	db.AddGuild(guildID, channelID)
}

func defaultTextChannel(s *discordgo.Session, chs []*discordgo.Channel) (int64, error) {
	for _, ch := range chs {
		if ch.Type != discordgo.ChannelTypeGuildText {
			continue
		}

		perms, err := s.State.UserChannelPermissions(s.State.User.ID, ch.ID)
		if err != nil {
			continue
		}

		if perms&discordgo.PermissionSendMessages == 0 {
			continue
		}

		channelID, err := strconv.ParseInt(ch.ID, 10, 64)
		if err != nil {
			continue
		}

		return channelID, nil
	}
	return 0, errors.New("no sendable text channel")
}

func (b *SteamBot) guildDeleteHandler(s *discordgo.Session, g *discordgo.GuildDelete) {
	// Do nothing if network outage, otherwise it means bot was removed from server
	if g.Unavailable {
		return
	}

	guildID, err := strconv.ParseInt(g.ID, 10, 64)
	if err != nil {
		return
	}

	db.RemoveGuild(guildID)
}

func (b *SteamBot) readyHandler(s *discordgo.Session, r *discordgo.Ready) {
	b.periodicallyUpdateStatus(s)
	b.periodicallyCheckApps(s)
}

func loc_PDT() *time.Location {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		log.Fatal("Failed to load location")
	}
	return loc
}

func nextCheck() time.Time {
	now := time.Now()
	timeOfCheck := time.Date(now.Year(), now.Month(), now.Day(), 10, 5, 0, 0, loc_PDT())
	if now.After(timeOfCheck) {
		timeOfCheck = timeOfCheck.Add(24 * time.Hour)
	}

	return timeOfCheck
}

func nextHour() time.Time {
	return time.Now().Add(time.Hour).Truncate(time.Hour)
}

func (b *SteamBot) periodicallyUpdateStatus(s *discordgo.Session) {
	var fn func()

	fn = func() {
		timeOfCheck := nextCheck()

		hrs := int(time.Until(timeOfCheck).Truncate(time.Hour).Hours())
		var plural string
		if hrs == 1 {
			plural = ""
		} else {
			plural = "s"
		}

		s.UpdateCustomStatus(fmt.Sprintf("%d hour%s until check", hrs, plural))

		time.AfterFunc(time.Until(nextHour()), fn)
	}

	fn()
}

func (b *SteamBot) periodicallyCheckApps(s *discordgo.Session) {
	var fn func()

	fn = func() {
		// TODO

		time.AfterFunc(time.Until(nextCheck()), fn)
	}

	fn()
}
