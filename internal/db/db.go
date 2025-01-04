// db provides database methods. Init() should be called before anything else.
// Close() should be called to close the database.
package db

import (
	"context"
	"errors"
	"fmt"
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

// Init intializes the database. Close() should be called to close the database.
func Init(uri, dbName string) {
	var err error

	client, err = mongo.Connect(context.Background(),
		options.Client().
			ApplyURI(uri).
			SetSocketTimeout(15*time.Second))
	if err != nil {
		log.Fatal("Failed to connect to db", err)
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal("Failed to ping db", err)
	}

	appsColl = client.Database(dbName).Collection("apps")
	discordColl = client.Database(dbName).Collection("discord")
	junctionColl = client.Database(dbName).Collection("junction")
}

// Close closes the database.
func Close() {
	if err := client.Disconnect(context.Background()); err != nil {
		log.Fatal("Disconnect error:", err)
	}
}

type AppRecord struct {
	Appid   int     `bson:"app_id"`
	AppName *string `bson:"app_name,omitempty"`
}

type DiscordRecord struct {
	ServerID      int64 `bson:"server_id,omitempty"`
	ChannelID     int64 `bson:"channel_id,omitempty"`
	SaleThreshold int   `bson:"sale_threshold,omitempty"`
}

type JunctionRecord struct {
	Appid           int   `bson:"app_id,omitempty"`
	ServerID        int64 `bson:"server_id,omitempty"`
	TrailingSaleDay *bool `bson:"is_trailing_sale_day,omitempty"`
}

func AddServer(guildID, channelID int64) error {
	if guildID == 0 || channelID == 0 {
		return errors.New("params cannot be 0. The omitempty tag will remove it from insertion")
	}

	filter := DiscordRecord{
		ServerID: guildID,
	}
	rec := bson.M{
		"$setOnInsert": DiscordRecord{
			ServerID:      guildID,
			ChannelID:     channelID,
			SaleThreshold: 1,
		}}

	_, err := discordColl.UpdateOne(
		context.Background(), filter, rec, options.Update().SetUpsert(true))
	return err
}

func AppsOf(guildID int64) ([]AppRecord, error) {
	juncfilter := JunctionRecord{ServerID: guildID}
	cur, err := junctionColl.Find(context.Background(), juncfilter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.Background())

	records := []AppRecord{}
	for cur.Next(context.Background()) {
		var juncRec JunctionRecord
		if err := cur.Decode(&juncRec); err != nil {
			continue
		}

		appFilter := AppRecord{Appid: juncRec.Appid}
		result := appsColl.FindOne(context.Background(), appFilter)
		if err := result.Err(); err != nil {
			continue
		}

		var appRec AppRecord
		if err := result.Decode(&appRec); err != nil {
			continue
		}

		records = append(records, appRec)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

// AddApps adds new appids to the apps collection and
// adds a junction of the appid and guildID to the junction collection.
// Lists of successfully added and failed to added apps are returned.
func AddApps(guildID int64, apps []steam.App) (succ []steam.App, fail []steam.App) {
	succ = []steam.App{}
	fail = []steam.App{}

	for i, app := range apps {
		// Start session
		sess, err := client.StartSession()
		if err != nil {
			return succ, append(fail, apps[i:]...)
		}
		defer sess.EndSession(context.Background())

		// Start transaction
		_, err = sess.WithTransaction(context.Background(),
			func(ctx mongo.SessionContext) (interface{}, error) {
				// Add to apps collection
				appFilter := AppRecord{Appid: app.Appid}
				appRec := bson.M{"$set": AppRecord{Appid: app.Appid, AppName: &app.Name}}

				_, err := appsColl.UpdateOne(
					ctx, appFilter, appRec, options.Update().SetUpsert(true))
				if err != nil {
					return nil, err
				}

				// Add to junction collection
				juncFilter := JunctionRecord{Appid: app.Appid, ServerID: guildID}
				trailing := false
				juncRec := bson.M{
					"$setOnInsert": JunctionRecord{
						Appid:           app.Appid,
						ServerID:        guildID,
						TrailingSaleDay: &trailing,
					}}

				_, err = junctionColl.UpdateOne(
					ctx, juncFilter, juncRec, options.Update().SetUpsert(true))
				if err != nil {
					return nil, err
				}

				return nil, nil
			},
		)

		if err != nil {
			fmt.Println("db err: ", err)
			fail = append(fail, app)
		} else {
			succ = append(succ, app)
		}
	}

	return succ, fail
}

func RemoveApps(guildID int64, appids []int) (succ []int, fail []int) {
	succ = []int{}
	fail = []int{}

	// Start session
	sess, err := client.StartSession()
	if err != nil {
		return succ, appids
	}
	defer sess.EndSession(context.Background())

	for _, appid := range appids {
		// Start transaction
		_, err = sess.WithTransaction(context.Background(),
			func(ctx mongo.SessionContext) (interface{}, error) {
				// Remove from junctions collection
				delFilter := JunctionRecord{Appid: appid, ServerID: guildID}
				_, err := junctionColl.DeleteOne(context.Background(), delFilter)
				if err != nil {
					return nil, err
				}

				// Remove from apps collection if last tracking guild
				findFilter := JunctionRecord{Appid: appid}
				count, err := junctionColl.CountDocuments(context.Background(), findFilter)
				if err != nil {
					return nil, err
				}

				// if not last guild tracking app, no need to cleanup in apps collection
				if count > 0 {
					return nil, nil
				}

				_, err = appsColl.DeleteOne(context.Background(), AppRecord{Appid: appid})
				if err != nil {
					return nil, err
				}

				return nil, nil
			},
		)

		if err != nil {
			fmt.Println("db err: ", err)
			fail = append(fail, appid)
		} else {
			succ = append(succ, appid)
		}
	}

	return succ, fail
}

func ClearApps(guildID int64) error {
	// Find junction records with guildID filter
	cur, err := junctionColl.Find(context.Background(), JunctionRecord{ServerID: guildID})
	if err != nil {
		return err
	}
	defer cur.Close(context.Background())

	// Transform returned records into a list of ids
	appids := []int{}
	for cur.Next(context.Background()) {
		var rec JunctionRecord
		if err := cur.Decode(&rec); err != nil {
			return err
		}

		appids = append(appids, rec.Appid)
	}
	if err := cur.Err(); err != nil {
		return err
	}

	_, fail := RemoveApps(guildID, appids)
	if len(fail) > 0 {
		return errors.New("failed to clear some apps")
	}

	return nil
}

func SetChannelID(guildID, channelID int64) error {
	if guildID == 0 || channelID == 0 {
		return errors.New("params cannot be 0. The omitempty tag will remove it from insertion")
	}

	filter := DiscordRecord{ServerID: guildID}
	rec := bson.M{"$set": DiscordRecord{ChannelID: channelID}}

	_, err := discordColl.UpdateOne(context.Background(), filter, rec)
	return err
}

func SetThreshold(guildID int64, threshold int) error {
	if guildID == 0 || threshold == 0 {
		return errors.New("params cannot be 0. The omitempty tag will remove it from insertion")
	}

	filter := DiscordRecord{ServerID: guildID}
	rec := bson.M{"$set": DiscordRecord{SaleThreshold: threshold}}

	_, err := discordColl.UpdateOne(context.Background(), filter, rec)
	return err
}
