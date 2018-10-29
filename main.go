package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/marni/goigc"
	"gopkg.in/mgo.v2/bson"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

/*
var tracks map[string]igc.Track
var ids IDArray
*/
var start time.Time
var meta Meta
var db MongoDB
var webhookDB MongoDB
var noOfTracks int

func handlerAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// return metadata for api
		http.Header.Add(w.Header(), "content-type", "application/json")
		elapsed := time.Since(start)
		meta.Uptime = fmt.Sprintf(
			"P%dDT%dH%dM%dS",
			int(elapsed.Seconds()/86400),
			int(elapsed.Hours())%24,
			int(elapsed.Minutes())%60,
			int(elapsed.Seconds())%60,
		)
		json.NewEncoder(w).Encode(meta)
	} else {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}
}

func handlerTrack(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	switch r.Method {
	case "POST":
		processingStart := time.Now()
		// get igc from url, add the track and return id
		http.Header.Add(w.Header(), "content-type", "application/json")

		var urlReq URLRequest
		decoder := json.NewDecoder(r.Body)
		decoder.Decode(&urlReq)

		var err error
		igcTrack, err := igc.ParseLocation(urlReq.URL)
		if err == nil {
			// add new track
			//id := fmt.Sprintf("test_track")

			igcTrack.Task.Start = igcTrack.Points[0]
			igcTrack.Task.Finish = igcTrack.Points[len(igcTrack.Points)-1]
			igcTrack.Task.Turnpoints = igcTrack.Points[1 : len(igcTrack.Points)-2]

			objectID := bson.NewObjectId()
			idString := ID{objectID.Hex()}
			timestamp := int64(time.Now().UnixNano() / 1000000)
			track := Track{
				objectID,
				igcTrack.Header.Date,
				igcTrack.Header.Pilot,
				igcTrack.Header.GliderType,
				igcTrack.Header.GliderID,
				igcTrack.Task.Distance(),
				urlReq.URL,
				idString.ID,
				timestamp,
			}
			db.Add(track)

			json.NewEncoder(w).Encode(idString)

			newTrackRegistrations := webhookDB.GetAllNewTrackRegistrations()
			tracks := db.GetAll()

			for i := 0; i < webhookDB.Count(); i++ {
				if db.Count()%newTrackRegistrations[i].MinTriggerValue == 0 {
					var trackIDs []string
					for j := db.Count() - 1; j > db.Count()-1-newTrackRegistrations[i].MinTriggerValue; j-- {
						if j < 0 {
							break
						}
						trackIDs = append(trackIDs, tracks[j].TrackID)
					}
					str := fmt.Sprintf(
						`{"content":"Latest timestamp: %d, new tracks: %v, processing: %fs"}`,
						timestamp,
						trackIDs,
						time.Since(processingStart).Seconds(),
					)
					req, err := http.NewRequest(
						"POST",
						newTrackRegistrations[i].WebhookURL,
						bytes.NewBufferString(str),
					)
					if err != nil {
						http.Error(w, http.StatusText(500), 500)
						return
					}
					req.Header.Set("content-type", "application/json")
					client := &http.Client{}
					resp, err := client.Do(req)
					if err != nil {
						http.Error(w, http.StatusText(500), 500)
						return
					}
					defer resp.Body.Close()
				}
			}

		} else {
			// malformed content
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
	case "GET":
		if len(parts) > 5 {
			// get track by id
			track, ok := db.Get(parts[4])
			if !ok {
				// id not found
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}

			if len(parts) == 6 {
				// return all metadata
				http.Header.Add(w.Header(), "content-type", "application/json")
				json.NewEncoder(w).Encode(track)
			} else if len(parts) == 7 {
				// return specific metadata
				switch parts[5] {
				case "pilot":
					fmt.Fprintf(w, track.Pilot)
				case "glider":
					fmt.Fprintf(w, track.Glider)
				case "glider_id":
					fmt.Fprintf(w, track.GliderID)
				case "track_length":
					fmt.Fprintf(w, "%f", track.TrackLength)
				case "H_date":
					fmt.Fprintf(w, "%v", track.HDate)
				case "track_src_url":
					fmt.Fprintf(w, track.TrackSrcURL)
				default:
					http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				}
			} else {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			}
		} else {
			// return all ids
			http.Header.Add(w.Header(), "content-type", "application/json")
			ids := IDContainer{}
			tracks := db.GetAll()
			for i := 0; i < len(tracks); i++ {
				ids.IDs = append(ids.IDs, tracks[i].TrackID)
			}
			json.NewEncoder(w).Encode(ids)
		}
	default:
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}
}

func handlerTickerLatest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tracks := db.GetAll()
		fmt.Fprintf(w, "%d", tracks[len(tracks)-1].Timestamp)
	} else {
		http.Error(w, http.StatusText(400), 400)
	}
}

func handlerTicker(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) == 5 {
			processingStart := time.Now()
			http.Header.Add(w.Header(), "content-type", "application/json")
			tracks := db.GetAll()

			if len(tracks) == 0 {
				return
			}

			ticker := Ticker{}
			for i := 0; i < 5; i++ { // TODO: this 5 is hardcoded (it shouldn't be)
				if len(tracks) > i {
					ticker.Tracks[i] = tracks[i].TrackID
				}
			}
			// alright, i know this isn't how this is supposed to work
			// but i honest-to-god-ly don't how i'm supposed to respond
			// without just using the stuff i have lying around in the
			// database, so yeah
			ticker.TStart = tracks[0].Timestamp
			ticker.TStop = tracks[int(math.Min(4, float64(len(tracks)-1)))].Timestamp
			ticker.TLatest = tracks[len(tracks)-1].Timestamp
			ticker.Processing = int(time.Since(processingStart).Seconds() * 1000)
			json.NewEncoder(w).Encode(ticker)
		} else if len(parts) == 6 {
			processingStart := time.Now()
			http.Header.Add(w.Header(), "content-type", "application/json")
			ts, err := strconv.Atoi(parts[4])
			timestamp := int64(ts)

			if err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			tracks := db.GetAll()
			var pagedTracks [5]Track

			if len(tracks) == 0 {
				return
			}

			ticker := Ticker{}

			currentTrackIndex := 0
			for tracks[currentTrackIndex].Timestamp <= timestamp || currentTrackIndex == len(tracks)-1 {
				currentTrackIndex++
			}

			for i := 0; i < 5; i++ {
				if currentTrackIndex < len(tracks) {
					pagedTracks[i] = tracks[currentTrackIndex+i]
					ticker.Tracks[i] = tracks[currentTrackIndex+i].TrackID
				}
			}

			ticker.TStart = pagedTracks[0].Timestamp
			ticker.TStop = tracks[currentTrackIndex-1].Timestamp
			ticker.TLatest = tracks[len(tracks)-1].Timestamp
			ticker.Processing = int(time.Since(processingStart).Seconds() * 1000)
			json.NewEncoder(w).Encode(ticker)
		} else {
			http.Error(w, http.StatusText(404), 404)
		}
	} else {
		http.Error(w, http.StatusText(400), 400)
	}
}

func handlerWebhookNewTrack(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		parts := strings.Split(r.URL.Path, "/")
		fmt.Println(parts[5])
		if len(parts) < 7 {
			var newTrackRegistration NewTrackRegistration
			decoder := json.NewDecoder(r.Body)
			decoder.Decode(&newTrackRegistration)

			objectID := bson.NewObjectId()
			idString := ID{objectID.Hex()}

			newTrackRegistration.ID = objectID
			newTrackRegistration.WebhookID = idString.ID
			webhookDB.AddNewTrackRegistration(newTrackRegistration)
		} else {
			if r.Method == "GET" {
				http.Header.Add(w.Header(), "content-type", "application/json")
				webhook, ok := webhookDB.GetWebhook(parts[5])
				if !ok {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)

				}
				json.NewEncoder(w).Encode(webhook)
			} else if r.Method == "DELETE" {
				http.Header.Add(w.Header(), "content-type", "application/json")
				webhook, ok := webhookDB.GetWebhook(parts[5])
				ok = webhookDB.DeleteWebhook(parts[5])
				if !ok {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				}
				json.NewEncoder(w).Encode(webhook)
			}
		}
	} else {
		http.Error(w, http.StatusText(400), 400)
	}
}

func handlerAdminAPITracksCount(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		fmt.Fprintf(w, "%d", db.Count())
	} else {
		http.Error(w, http.StatusText(400), 400)
	}
}

func handlerAdminAPITracks(w http.ResponseWriter, r *http.Request) {
	if r.Method == "DELETE" {
		tracks := db.GetAll()
		var trackIDs []string
		maxI := len(tracks)
		for i := 0; i < maxI; i++ {
			trackIDs = append(trackIDs, tracks[i].TrackID)
		}
		for i := 0; i < maxI; i++ {
			db.Delete(trackIDs[i])
		}
		fmt.Fprintf(w, "%d", maxI)
	} else {
		http.Error(w, http.StatusText(400), 400)
	}
}

func determineListenAddress() string {
	port := os.Getenv("PORT")
	return ":" + port
}

func clockTrigger() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			currNoOfTracks := len(db.GetAll())
			if currNoOfTracks != noOfTracks {
				noOfTracks = currNoOfTracks
				str := []byte(`{"content":"wowee"}`)
				req, err := http.NewRequest(
					"POST",
					"https://discordapp.com/api/webhooks/506425503163482122/cLEX2L9E_ArBvG-JTstwsLXeA5uKsgHHQvH8FgaZA8RyTYobdpLXduNBJRtIBNJ1pgIj",
					bytes.NewBuffer(str),
				)
				req.Header.Set("content-type", "application/json")
				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					panic(err)
				}
				defer resp.Body.Close()
			}
		}
	}()
}

func main() {
	go clockTrigger()
	db = MongoDB{"mongodb://admin:admin1@ds141783.mlab.com:41783/trackdb", "trackdb", "trackcollection"}
	webhookDB = MongoDB{"mongodb://admin:admin1@ds141783.mlab.com:41783/trackdb", "trackdb", "webhookcollection"}
	db.Init()
	webhookDB.InitWebhook()

	start = time.Now()

	meta.Info = "Service for Paragliding tracks."
	meta.Version = "v1"

	noOfTracks = len(db.GetAll())

	http.HandleFunc("/paragliding/api/", handlerAPI)
	http.HandleFunc("/paragliding/api/track/", handlerTrack)
	http.HandleFunc("/paragliding/api/ticker/", handlerTicker)
	http.HandleFunc("/paragliding/api/ticker/latest/", handlerTickerLatest)
	http.HandleFunc("/paragliding/api/webhook/new_track/", handlerWebhookNewTrack)
	http.HandleFunc("/paragliding/admin/api/tracks_count/", handlerAdminAPITracksCount)
	http.HandleFunc("/paragliding/admin/api/tracks/", handlerAdminAPITracks)
	http.ListenAndServe(determineListenAddress(), nil)
}
