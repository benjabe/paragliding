package main

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"time"
)

// MongoDB stores the details of the DB connection.
type MongoDB struct {
	DatabaseURL    string
	DatabaseName   string
	CollectionName string
}

type Meta struct {
	Uptime  string `json:"uptime"`
	Info    string `json:"info"`
	Version string `json:"version"`
}

type URLRequest struct {
	URL string `json:"url"`
}

type Track struct {
	Id          bson.ObjectId `bson:"_id,omitempty"`
	HDate       time.Time     `json:"H_date"`
	Pilot       string        `json:"pilot"`
	Glider      string        `json:"glider"`
	GliderID    string        `json:"glider_id"`
	TrackLength float64       `json:"track_length"`
	TrackSrcURL string        `json:"track_src_url"`
	TrackID     string        `json:"trackid"`
	Timestamp   int64         `json:"timestamp"`
}

type NewTrackRegistration struct {
	Id              bson.ObjectId `bson:"_id,omitempty"`
	WebhookID       string        `json:"webhookid"`
	WebhookURL      string        `json:"webhookurl"`
	MinTriggerValue int           `json:"minTriggerValue"`
}

type ID struct {
	ID string `json:id`
}

type IDContainer struct {
	IDs []string `json:ids`
}

/*
Init initializes the mongo storage.
*/
func (db *MongoDB) Init() {
	session, err := mgo.Dial(db.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	index := mgo.Index{
		Key:        []string{"trackid"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}

	err = session.DB(db.DatabaseName).C(db.CollectionName).EnsureIndex(index)
	if err != nil {
		panic(err)
	}
}

func (db *MongoDB) InitWebhook() {
	session, err := mgo.Dial(db.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	index := mgo.Index{
		Key:        []string{"webhookid"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}

	err = session.DB(db.DatabaseName).C(db.CollectionName).EnsureIndex(index)
	if err != nil {
		panic(err)
	}
}

/*
Add adds new tracks to the storage.
*/
func (db *MongoDB) Add(t Track) error {
	session, err := mgo.Dial(db.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	err = session.DB(db.DatabaseName).C(db.CollectionName).Insert(t)

	if err != nil {
		fmt.Printf("error in Insert(): %v", err.Error())
		return err
	}
	return nil
}

func (db *MongoDB) AddNewTrackRegistration(n NewTrackRegistration) error {
	session, err := mgo.Dial(db.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	err = session.DB(db.DatabaseName).C(db.CollectionName).Insert(n)

	if err != nil {
		fmt.Printf("error in Insert(): %v", err.Error())
		return err
	}
	return nil
}

/*
Count returns the current count of the tracks in in-memory storage.
*/
func (db *MongoDB) Count() int {
	session, err := mgo.Dial(db.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// handle to "db"
	count, err := session.DB(db.DatabaseName).C(db.CollectionName).Count()
	if err != nil {
		fmt.Printf("error in Count(): %v", err.Error())
		return -1
	}

	return count
}

/*
Get returns a track with a given ID or empty student struct.
*/
func (db *MongoDB) Get(keyID string) (Track, bool) {
	session, err := mgo.Dial(db.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	track := Track{}
	allWasGood := true

	err = session.DB(db.DatabaseName).C(db.CollectionName).Find(bson.M{"trackid": keyID}).One(&track)
	if err != nil {
		allWasGood = false
	}

	return track, allWasGood
}

func (db *MongoDB) GetWebhook(url string) (NewTrackRegistration, bool) {
	session, err := mgo.Dial(db.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	n := NewTrackRegistration{}
	allWasGood := true

	err = session.DB(db.DatabaseName).C(db.CollectionName).Find(bson.M{"webhookid": url}).One(&n)
	if err != nil {
		allWasGood = false
	}

	return n, allWasGood
}

/*
GetAll returns a slice with all the tracks.
*/
func (db *MongoDB) GetAll() []Track {
	session, err := mgo.Dial(db.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	var all []Track

	err = session.DB(db.DatabaseName).C(db.CollectionName).Find(bson.M{}).All(&all)
	if err != nil {
		return []Track{}
	}

	return all
}

func (db *MongoDB) GetAllNewTrackRegistrations() []NewTrackRegistration {
	session, err := mgo.Dial(db.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	var all []NewTrackRegistration

	err = session.DB(db.DatabaseName).C(db.CollectionName).Find(bson.M{}).All(&all)
	if err != nil {
		return []NewTrackRegistration{}
	}

	return all
}

func (db *MongoDB) Delete(keyID string) bool {
	session, err := mgo.Dial(db.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	ok := true

	err = session.DB(db.DatabaseName).C(db.CollectionName).Remove(bson.M{"trackid": keyID})
	if err != nil {
		ok = false
	}
	return ok
}

func (db *MongoDB) DeleteWebhook(keyID string) bool {
	session, err := mgo.Dial(db.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	ok := true

	err = session.DB(db.DatabaseName).C(db.CollectionName).Remove(bson.M{"webhookid": keyID})
	if err != nil {
		ok = false
	}
	return ok
}
