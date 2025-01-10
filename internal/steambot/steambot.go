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
	"github.com/jasonly027/steam_sale_discord_bot_go/internal/steam"
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
		guildCreateHandler,
		guildDeleteHandler,
		readyHandler,
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
		cmd.NewAddApps(),
		cmd.NewBind(),
		cmd.NewClearApps(),
		cmd.NewHelp(),
		cmd.NewListApps(),
		cmd.NewRemoveApps(),
		cmd.NewSearch(),
		cmd.NewSetDiscountThreshold(),
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
			return
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

func guildCreateHandler(s *discordgo.Session, g *discordgo.GuildCreate) {
	guildID, err := strconv.ParseInt(g.ID, 10, 64)
	if err != nil {
		return
	}

	channelID, err := defaultTextChannel(s, g.Channels)
	if err != nil {
		channelID = 0
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

func guildDeleteHandler(s *discordgo.Session, g *discordgo.GuildDelete) {
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

func readyHandler(s *discordgo.Session, r *discordgo.Ready) {
	periodicallyUpdateStatus(s)
	periodicallyCheckApps(s)
}

// periodicallyUpdateStatus will update the Discord status of the bot
// to the number of hours left until a sale check is done. Once called,
// it will call itself every whole hour.
func periodicallyUpdateStatus(s *discordgo.Session) {
	var fn func()

	fn = func() {
		hrs := int(time.Until(nextCheck()).Truncate(time.Hour).Hours())
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

func loc_PDT() *time.Location {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		log.Fatal("Failed to load location")
	}
	return loc
}

// nextCheck gets when the next sale check is as a Time object.
func nextCheck() time.Time {
	loc := loc_PDT()
	now := time.Now().In(loc)
	timeOfCheck := time.Date(now.Year(), now.Month(), now.Day(), 10, 5, 0, 0, loc)
	if now.After(timeOfCheck) {
		timeOfCheck = timeOfCheck.Add(24 * time.Hour)
	}

	return timeOfCheck
}

// nextHour gets when the next whole hour is as a Time object.
func nextHour() time.Time {
	return time.Now().Add(time.Hour).Truncate(time.Hour)
}

// periodicallyCheckApps will go through all globally added apps to the bot
// and send sale alerts to all the servers tracking that app if the sale discount
// is at least that server's discount threshold. Once called, it will call itself
// daily at 10:05 AM PDT. Due to external API rate limiting when getting app info,
// the time it takes to finish checking may take a while but not long enough to
// miss the next daily check. Calls to the external API are done until a rate limit
// is hit, then this fn waits a period before trying to continue.
func periodicallyCheckApps(s *discordgo.Session) {
	// This is the fn that will be periodically called to check apps for sales.
	var checkApps func()

	// This fn will attempt to fetch an App by appid and check for a sale,
	// If an error occurs fetching the App (like an inevitable rate limit error
	// from Steam), a wait out period will be put in place, then checkApps will
	// be called again to resume checking. Returns whether or not the calling fn
	// (checkApps) needs to exit for a cooldown.
	var tryCheckApp func(appid int) bool

	// The id of the App we are fetching. Populated through nextAppid().
	var currAppid *int

	// This fn will return appids we need to check. When there are no more
	// appids to check, this will only return nil.
	var nextAppid func() *int

	// This fn must be called to release resources when we're done calling
	// nextAppid().
	var close func()

	// This fn clears the state of checkApps so that the next call to it
	// is not considered a resuming check.
	reset := func() {
		close()
		currAppid = nil
		nextAppid = nil
	}

	checkApps = func() {
		if nextAppid == nil { // Get fresh apps if non-resuming check
			nextAppid, close = db.Apps()
		}

		// Not nil means we are resuming from the previous check that we
		// were rate limited on. This was the appid we failed to create
		// an App from, so we retry it.
		if currAppid != nil {
			if exit := tryCheckApp(*currAppid); exit {
				return
			}
		}

		currAppid = nextAppid()
		for currAppid != nil {
			if exit := tryCheckApp(*currAppid); exit {
				return
			}
			currAppid = nextAppid()
		}

		// At this point, we have checked all apps, now schedule tomorrow's check
		reset()
		time.AfterFunc(time.Until(nextCheck()), checkApps)
	}

	tryCheckApp = func(appid int) (exit bool) {
		app, err := steam.NewApp(appid)
		if err != nil {
			// On rate-limit, wait then resume
			if err == steam.ErrNetTryAgainLater {
				time.AfterFunc(5*time.Minute, checkApps)
				// On any other error, just abort today's check
			} else {
				reset()
				time.AfterFunc(time.Until(nextCheck()), checkApps)
			}
			return true
		}

		checkApp(s, app)
		return false
	}

	checkApps()
}

// checkApp goes through every guild tracking app and sends a sale alert
// to that guild if there is a sale discount that is at least equal to
// the server's discount threshold.
func checkApp(s *discordgo.Session, app steam.App) {
	guilds, err := db.GuildsOf(app.Appid)
	if err != nil {
		return
	}

	for _, guild := range guilds {
		updateGuildOnApp(s, app, guild)
	}
}

func updateGuildOnApp(s *discordgo.Session, app steam.App, guild db.GuildInfo) {
	db.SetTrailingSaleDay(guild.ServerID, guild.Appid, app.Discount > 0)
	db.SetComingSoon(guild.ServerID, guild.Appid, app.ComingSoon)

	channelID := strconv.FormatInt(guild.ChannelID, 10)

	if !app.ComingSoon && guild.ComingSoon {
		s.ChannelMessageSendEmbed(channelID, releaseEmbed(app))
	} else if app.Discount >= guild.SaleThreshold && !guild.TrailingSaleDay &&
		guild.ChannelID != 0 {
		s.ChannelMessageSendEmbed(channelID, saleEmbed(app))
	}
}

func releaseEmbed(app steam.App) *discordgo.MessageEmbed {
	fields := []*discordgo.MessageEmbedField{
		{
			Name:  "Price",
			Value: app.Final,
		},
		{
			Name:  "Description",
			Value: app.Description,
		},
	}
	if fields[0].Value == "" {
		fields[0].Value = "Free"
	}

	return &discordgo.MessageEmbed{
		Title:  fmt.Sprintf("%s has released on Steam!", app.Name),
		URL:    app.Url(),
		Image:  &discordgo.MessageEmbedImage{URL: app.Image},
		Color:  0xFFFFFF,
		Fields: fields,
	}
}

func saleEmbed(app steam.App) *discordgo.MessageEmbed {
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Original Price",
			Value:  app.Initial,
			Inline: true,
		},
		{
			Name:   "Sale Price",
			Value:  app.Final,
			Inline: true,
		},
	}
	if app.Reviews > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Reviews",
			Value:  strconv.Itoa(app.Reviews),
			Inline: true,
		})
	}
	if app.Description != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Description",
			Value: app.Description,
		})
	}

	return &discordgo.MessageEmbed{
		Title:  fmt.Sprintf("%s is on sale for %d%% off!", app.Name, app.Discount),
		URL:    app.Url(),
		Image:  &discordgo.MessageEmbedImage{URL: app.Image},
		Color:  discountColor(app.Discount),
		Fields: fields,
	}
}

func discountColor(discount int) int {
	atMost := func(high int) bool {
		return discount <= high
	}
	switch {
	case atMost(5):
		return 0x0bff33
	case atMost(10):
		return 0x44fdd2
	case atMost(15):
		return 0x44fdfd
	case atMost(20):
		return 0x44dbfd
	case atMost(25):
		return 0x44b6fd
	case atMost(30):
		return 0x448bfd
	case atMost(35):
		return 0x445afd
	case atMost(40):
		return 0x8544fd
	case atMost(45):
		return 0xb044fd
	case atMost(50):
		return 0xe144fd
	case atMost(55):
		return 0xfd44de
	case atMost(60):
		return 0xff23a7
	case atMost(99):
		return 0xff0000
	default:
		return 0xFFFFFF
	}
}
