// db provides database methods. Init() should be called before anything else.
// Close() should be called to close the database.
package db

import (
	"context"
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
	Appid int `bson:"app_id"`
}

type DiscordRecord struct {
	ServerID      int64 `bson:"server_id"`
	ChannelID     int64 `bson:"channel_id"`
	SaleThreshold int   `bson:"sale_threshold"`
}

type JunctionRecord struct {
	Appid           int   `bson:"app_id"`
	ServerID        int64 `bson:"server_id"`
	TrailingSaleDay bool  `bson:"is_trailing_sale_day"`
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
				appRec := AppRecord{
					Appid: app.Appid,
				}
				_, err := appsColl.UpdateOne(
					ctx, appRec, bson.M{"$setOnInsert": appRec}, options.Update().SetUpsert(true))
				if err != nil {
					return nil, err
				}

				// Add to junction collection
				juncRec := JunctionRecord{
					Appid:           app.Appid,
					ServerID:        guildID,
					TrailingSaleDay: false,
				}
				_, err = junctionColl.UpdateOne(
					ctx, juncRec, bson.M{"$setOnInsert": juncRec}, options.Update().SetUpsert(true))
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
