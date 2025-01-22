package db

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jasonly027/steam_sale_discord_bot_go/internal/steam"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

var (
	appsColl,
	discordColl,
	junctionColl *mongo.Collection
)

type AppRecord struct {
	Appid   *int    `bson:"app_id,omitempty"`
	AppName *string `bson:"app_name,omitempty"`
}

type AppInfo struct {
	Appid   int    `bson:"app_id"`
	AppName string `bson:"app_name"`
}

type DiscordRecord struct {
	ServerID      *int64 `bson:"server_id,omitempty"`
	ChannelID     *int64 `bson:"channel_id,omitempty"`
	SaleThreshold *int   `bson:"sale_threshold,omitempty"`
}

type DiscordInfo struct {
	ServerID      int64 `bson:"server_id"`
	ChannelID     int64 `bson:"channel_id"`
	SaleThreshold int   `bson:"sale_threshold"`
}

type JunctionRecord struct {
	Appid           *int   `bson:"app_id,omitempty"`
	ServerID        *int64 `bson:"server_id,omitempty"`
	TrailingSaleDay *bool  `bson:"is_trailing_sale_day,omitempty"`
	ComingSoon      *bool  `bson:"coming_soon,omitempty"`
	SaleThreshold   *int   `bson:"sale_threshold,omitempty"`
}

type JunctionInfo struct {
	Appid           int   `bson:"app_id"`
	ServerID        int64 `bson:"server_id"`
	TrailingSaleDay bool  `bson:"is_trailing_sale_day"`
	ComingSoon      bool  `bson:"coming_soon"`
	SaleThreshold   int   `bson:"sale_threshold"`
}

type GuildInfo struct {
	ServerID         int64
	ChannelID        int64
	Appid            int
	AppName          string
	AppSaleThreshold int
	SaleThreshold    int
	TrailingSaleDay  bool
	ComingSoon       bool
}

func ctx() context.Context {
	return context.Background()
}

// Init intializes the database.
// Close() should be called to close the database.
func Init(uri, dbName string) {
	var err error

	client, err = mongo.Connect(
		ctx(),
		options.Client().
			ApplyURI(uri).
			SetSocketTimeout(15*time.Second),
	)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	if err = client.Ping(context.Background(), nil); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	appsColl = client.Database(dbName).Collection("apps")
	discordColl = client.Database(dbName).Collection("discord")
	junctionColl = client.Database(dbName).Collection("junction")
}

// Close closes the database
func Close() {
	if err := client.Disconnect(context.Background()); err != nil {
		log.Fatal("Error disconnecting:", err)
	}
}

// validateColFilDoc verifies that the combination of parameters
// are meant to be used with each other. They should all be the
// same kind App, Discord, or Junction.
//
// Panics on invalid combinations or types.
func validateColFilDoc(coll *mongo.Collection, filter any, doc any) {
	var ok bool

	switch filter.(type) {
	case AppRecord:
		_, ok = doc.(AppRecord)
		ok = ok && (coll == appsColl)
	case DiscordRecord:
		_, ok = doc.(DiscordRecord)
		ok = ok && (coll == discordColl)
	case JunctionRecord:
		_, ok = doc.(JunctionRecord)
		ok = ok && (coll == junctionColl)
	}

	if !ok {
		panic("coll, filter, doc are invalid types or type mismatch")
	}
}

// insert adds a doc to coll if there is nothing in coll matching filter.
// If there is a doc matching filter, no insertion occurs and it isn't
// considered an error.
func insert(coll *mongo.Collection, filter any, doc any) error {
	validateColFilDoc(coll, filter, doc)
	_, err := coll.UpdateOne(
		ctx(),
		filter,
		bson.M{
			"$setOnInsert": doc,
		},
		options.Update().SetUpsert(true),
	)
	return err
}

// update finds a doc in coll matching filter and updates it with doc.
// If there is no doc matching filter, no update occurs and it isn't
// considered an error.
func update(coll *mongo.Collection, filter any, doc any) error {
	validateColFilDoc(coll, filter, doc)
	_, err := coll.UpdateOne(
		ctx(),
		filter,
		bson.M{
			"$set": doc,
		},
	)
	return err
}

// upsert finds a doc in coll matching filter and updates it with doc.
// If there is no doc matching filter, doc is added.
func upsert(coll *mongo.Collection, filter any, doc any) error {
	validateColFilDoc(coll, filter, doc)
	_, err := coll.UpdateOne(
		ctx(),
		filter,
		bson.M{
			"$set": doc,
		},
		options.Update().SetUpsert(true),
	)
	return err
}

// AddGuild adds a new guild to the database by its guildID. If there is
// already a record in the database with the same guildID, nothing happens
// and it isn't considered an error. If a channelID cannot be added right now,
// pass 0 for channelID.
func AddGuild(guildID, channelID int64) error {
	saleThreshold := 1
	return insert(discordColl,
		DiscordRecord{ServerID: &guildID},
		DiscordRecord{
			ServerID:      &guildID,
			ChannelID:     &channelID,
			SaleThreshold: &saleThreshold,
		},
	)
}

// RemoveGuild removes a guild and its app from the database.
// If there's no record in the database with the guildID, nothing happens
// and it isn't considered an error.
func RemoveGuild(guildID int64) error {
	ClearApps(guildID)
	_, err := discordColl.DeleteOne(ctx(),
		DiscordRecord{ServerID: &guildID})
	return err
}

// AppsOf finds all GuildInfos tracked by the guild matching guildID.
// If guildID hasn't been added through AddGuild(...), an empty
// list will be returned.
func AppsOf(guildID int64) (guildInfos []GuildInfo, err error) {
	findGuildRes := discordColl.FindOne(ctx(), DiscordRecord{ServerID: &guildID})
	if err := findGuildRes.Err(); err != nil {
		return nil, err
	}
	var dInfo DiscordInfo
	if err := findGuildRes.Decode(&dInfo); err != nil {
		return nil, err
	}

	cur, err := junctionColl.Find(ctx(), JunctionRecord{ServerID: &guildID})
	if err != nil {
		return nil, err
	}

	// Extract appid from each JunctionRecord to filter
	// for that App's AppRecord
	for cur.Next(ctx()) {
		var jInfo JunctionInfo
		if err := cur.Decode(&jInfo); err != nil {
			continue
		}

		findAppRes := appsColl.FindOne(ctx(), AppRecord{Appid: &jInfo.Appid})
		if err := findAppRes.Err(); err != nil {
			continue
		}

		var aInfo AppInfo
		if err := findAppRes.Decode(&aInfo); err != nil {
			continue
		}

		guildInfos = append(guildInfos,
			GuildInfo{
				ServerID:         dInfo.ServerID,
				ChannelID:        dInfo.ChannelID,
				Appid:            aInfo.Appid,
				AppName:          aInfo.AppName,
				AppSaleThreshold: jInfo.SaleThreshold,
				SaleThreshold:    dInfo.SaleThreshold,
				TrailingSaleDay:  jInfo.TrailingSaleDay,
				ComingSoon:       jInfo.ComingSoon,
			})
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	return guildInfos, nil
}

func Apps() (nextApp func() *int, close func()) {
	cur, err := appsColl.Find(ctx(), bson.M{})
	if err != nil {
		return func() *int { return nil }, func() {}
	}

	nextApp = func() *int {
		for cur.Next(ctx()) {
			var rec AppInfo
			err := cur.Decode(&rec)
			if err != nil {
				continue
			}
			return &rec.Appid
		}
		return nil
	}

	close = func() {
		cur.Close(ctx())
	}

	return nextApp, close
}

// GuildsOf finds all guilds tracking the app specified by appid.
// If appid wasn't added through AddApps(...), guildInfos will be empty.
func GuildsOf(appid int) (guildInfos []GuildInfo, err error) {
	cur, err := junctionColl.Find(ctx(), JunctionRecord{Appid: &appid})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx())

	for cur.Next(ctx()) {
		var jInfo JunctionInfo
		if err := cur.Decode(&jInfo); err != nil {
			continue
		}

		// Given the guildID from a junction, find more info on the guild in
		// the Discord collection to be able to initialize every field in the
		// GuildInfo object
		findDiscRes := discordColl.FindOne(ctx(), DiscordRecord{ServerID: &jInfo.ServerID})
		if err := findDiscRes.Err(); err != nil {
			continue
		}

		var dInfo DiscordInfo
		if err := findDiscRes.Decode(&dInfo); err != nil {
			continue
		}

		guildInfos = append(guildInfos,
			GuildInfo{
				ServerID:         dInfo.ServerID,
				ChannelID:        dInfo.ChannelID,
				Appid:            jInfo.Appid,
				AppSaleThreshold: jInfo.SaleThreshold,
				SaleThreshold:    dInfo.SaleThreshold,
				TrailingSaleDay:  jInfo.TrailingSaleDay,
				ComingSoon:       jInfo.ComingSoon,
			},
		)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	return guildInfos, nil
}

// AddApps adds apps under a guild. If guildID hasn't been added through AddGuild(...),
// adding the apps will still work but they won't be retrievable through AppsOf(...).
func AddApps(guildID int64, apps []*steam.App) (succ []*steam.App, fail []*steam.App) {
	sess, err := client.StartSession()
	if err != nil {
		return nil, apps
	}
	defer sess.EndSession(ctx())

	// For each app, attempt the transaction
	// of upserting an App and inserting a Junction
	for _, app := range apps {
		transactionFn := func(ctx mongo.SessionContext) (any, error) {
			err := upsert(appsColl,
				AppRecord{Appid: &app.Appid},
				AppRecord{
					Appid:   &app.Appid,
					AppName: &app.Name, // Upsertion is done because name may have changed
				},
			)
			if err != nil {
				return nil, err
			}

			trailingSaleDay := false
			err = insert(junctionColl,
				JunctionRecord{Appid: &app.Appid, ServerID: &guildID},
				JunctionRecord{
					Appid:           &app.Appid,
					ServerID:        &guildID,
					TrailingSaleDay: &trailingSaleDay,
					ComingSoon:      &app.ComingSoon,
					SaleThreshold:   app.SaleThreshold,
				},
			)
			if err != nil {
				return nil, err
			}

			return nil, nil
		}

		if _, err := sess.WithTransaction(ctx(), transactionFn); err != nil {
			fail = append(fail, app)
		} else {
			succ = append(succ, app)
		}
	}

	return succ, fail
}

// RemoveApps removes apps from a guild. If an appid from appids isn't
// actually under this guild, the removal is still considered successful
// and placed in the succ list.
func RemoveApps(guildID int64, appids []int) (succ []int, fail []int) {
	sess, err := client.StartSession()
	if err != nil {
		return nil, appids
	}
	defer sess.EndSession(context.Background())

	// For each app, attempt the transaction of
	// removing the JunctionRecord and removing the AppRecord if
	// the JunctionRecord was the last junction referencing it
	for _, appid := range appids {
		transactionFn := func(ctx mongo.SessionContext) (any, error) {
			_, err := junctionColl.DeleteOne(ctx,
				JunctionRecord{Appid: &appid, ServerID: &guildID})
			if err != nil {
				return nil, err
			}

			count, err := junctionColl.CountDocuments(ctx, JunctionRecord{Appid: &appid})
			if err != nil {
				return nil, err
			} else if count > 0 { // If not an orphan, no need to remove
				return nil, nil
			}

			_, err = appsColl.DeleteOne(ctx, AppRecord{Appid: &appid})
			if err != nil {
				return nil, err
			}

			return nil, nil
		}

		if _, err := sess.WithTransaction(ctx(), transactionFn); err != nil {
			fail = append(fail, appid)
		} else {
			succ = append(succ, appid)
		}
	}

	return succ, fail
}

// ClearApps clears the apps under guildID. Does nothing if there
// are no apps under the guild.
func ClearApps(guildID int64) error {
	cur, err := junctionColl.Find(ctx(), JunctionRecord{ServerID: &guildID})
	if err != nil {
		return err
	}
	defer cur.Close(ctx())

	// Extract appids
	appids := []int{}
	for cur.Next(ctx()) {
		var rec JunctionInfo
		if err := cur.Decode(&rec); err != nil {
			continue
		}

		appids = append(appids, rec.Appid)
	}
	if err := cur.Err(); err != nil {
		return err
	}

	if _, fail := RemoveApps(guildID, appids); len(fail) > 0 {
		return errors.New("failed to clear some apps")
	}

	return nil
}

// SetChannelID sets the channelID alerts are sent for a guild
func SetChannelID(guildID, channelID int64) error {
	return update(discordColl,
		DiscordRecord{ServerID: &guildID},
		DiscordRecord{ChannelID: &channelID},
	)
}

// SetThreshold sets the sale threshold for alerts sent to a guild
func SetThreshold(guildID int64, threshold int) error {
	return update(discordColl,
		DiscordRecord{ServerID: &guildID},
		DiscordRecord{SaleThreshold: &threshold},
	)
}

// SetThresholds sets the sale threshold for alerts sent to a guild
// for the specific appids
func SetThresholds(guildID int64, threshold int, appids []int) (succ []int, fail []int) {
	sess, err := client.StartSession()
	if err != nil {
		return nil, appids
	}
	defer sess.EndSession(ctx())

	for _, appid := range appids {
		transactionFn := func(ctx mongo.SessionContext) (any, error) {
			err := update(junctionColl,
				JunctionRecord{ServerID: &guildID, Appid: &appid},
				JunctionRecord{SaleThreshold: &threshold})
			if err != nil {
				return nil, err
			}

			return nil, nil
		}

		if _, err := sess.WithTransaction(ctx(), transactionFn); err != nil {
			fail = append(fail, appid)
		} else {
			succ = append(succ, appid)
		}
	}

	return succ, fail
}

// SetTrailingSaleDay sets the trailing sale day field for an app for a guild
func SetTrailingSaleDay(guildID int64, appid int, sale bool) error {
	return update(junctionColl,
		JunctionRecord{ServerID: &guildID, Appid: &appid},
		JunctionRecord{TrailingSaleDay: &sale},
	)
}

// SetComingSoon sets the coming soon field for an app for a guild
func SetComingSoon(guildID int64, appid int, comingSoon bool) error {
	return update(junctionColl,
		JunctionRecord{ServerID: &guildID, Appid: &appid},
		JunctionRecord{ComingSoon: &comingSoon},
	)
}
