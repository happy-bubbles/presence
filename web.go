package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/boltdb/bolt"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// GLOBALS

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func startServer() {
	// Set up HTTP server
	r := mux.NewRouter()
	r.HandleFunc("/api/results", resultsHandler)

	r.HandleFunc("/api/beacons/{beacon_id}", beaconsDeleteHandler).Methods("DELETE")
	r.HandleFunc("/api/beacons", beaconsListHandler).Methods("GET")
	r.HandleFunc("/api/beacons", beaconsAddHandler).Methods("POST") //since beacons are hashmap, just have put and post be same thing. it'll either add or modify that entry
	r.HandleFunc("/api/beacons", beaconsAddHandler).Methods("PUT")

	r.HandleFunc("/api/latest-beacons", latestBeaconsListHandler).Methods("GET")

	// what should this be?
	// probably all the current command-line things:
	// * mqtt connect stuff user/pass/host/port/client-id - have indicator to show it's connected to mqtt server, reconnect when needed
	// * thresholds
	r.HandleFunc("/api/settings", settingsListHandler).Methods("GET")
	r.HandleFunc("/api/settings", settingsEditHandler).Methods("POST")

	r.PathPrefix("/js/").Handler(http.StripPrefix("/js/", http.FileServer(http.Dir("static_html/js/"))))
	r.PathPrefix("/css/").Handler(http.StripPrefix("/css/", http.FileServer(http.Dir("static_html/css/"))))
	r.PathPrefix("/img/").Handler(http.StripPrefix("/img/", http.FileServer(http.Dir("static_html/img/"))))
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("static_html/")))

	http.Handle("/", r)
	http.HandleFunc("/api/beacons/ws", serveWs)
	http.HandleFunc("/api/beacons/latest/ws", serveLatestBeaconsWs)
	log.Fatal(http.ListenAndServe(*http_host_path_ptr, nil))
}

func resultsHandler(w http.ResponseWriter, r *http.Request) {
	http_results_lock.RLock()
	js, err := json.Marshal(http_results)
	http_results_lock.RUnlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(js)
}

func beaconsListHandler(w http.ResponseWriter, r *http.Request) {
	latest_list_lock.RLock()
	js, err := json.Marshal(BEACONS)
	latest_list_lock.RUnlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(js)
}

func persistBeacons() error {
	// gob it first
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(BEACONS); err != nil {
		return err
	}

	key := []byte("beacons_list")
	// store some data
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(world)
		if err != nil {
			return err
		}

		err = bucket.Put(key, []byte(buf.String()))
		if err != nil {
			return err
		}
		return nil
	})
	return nil
}

func persistButtons() error {
	// gob it first
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(Buttons_list); err != nil {
		return err
	}

	key := []byte("buttons_list")
	// store some data
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(world)
		if err != nil {
			return err
		}

		err = bucket.Put(key, []byte(buf.String()))
		if err != nil {
			return err
		}
		return nil
	})
	return nil
}

func persistSettings() error {
	// gob it first
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(settings); err != nil {
		return err
	}

	key := []byte("settings")
	// store some data
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(world)
		if err != nil {
			return err
		}

		err = bucket.Put(key, []byte(buf.String()))
		if err != nil {
			return err
		}
		return nil
	})
	return nil
}

func beaconsAddHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var in_beacon Beacon
	err = decoder.Decode(&in_beacon)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	//make sure name and beacon_id are present
	if (len(strings.TrimSpace(in_beacon.Name)) == 0) || (len(strings.TrimSpace(in_beacon.Beacon_id)) == 0) {
		http.Error(w, "name and beacon_id cannot be blank", 400)
		return
	}

	BEACONS.Beacons[in_beacon.Beacon_id] = in_beacon

	err := persistBeacons()
	if err != nil {
		http.Error(w, "trouble persisting beacons list, create bucket", 500)
		return
	}

	w.Write([]byte("ok"))
}

func beaconsDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	beacon_id := vars["beacon_id"]
	delete(BEACONS.Beacons, beacon_id)

	_, ok := Buttons_list[beacon_id]
	if ok {
		delete(Buttons_list, beacon_id)
	}

	err := persistBeacons()
	if err != nil {
		http.Error(w, "trouble persisting beacons list, create bucket", 500)
		return
	}

	w.Write([]byte("ok"))
}

func latestBeaconsListHandler(w http.ResponseWriter, r *http.Request) {
	latest_list_lock.RLock()
	var la = make([]Beacon, 0)
	for _, b := range Latest_beacons_list {
		la = append(la, b)
	}
	latest_list_lock.RUnlock()
	js, err := json.Marshal(la)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(js)
}

func settingsListHandler(w http.ResponseWriter, r *http.Request) {
	js, err := json.Marshal(settings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(js)
}

func settingsEditHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var in_settings Settings
	err = decoder.Decode(&in_settings)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	//make sure values are > 0
	if (in_settings.Location_confidence <= 0) ||
		(in_settings.Last_seen_threshold <= 0) ||
		(in_settings.HA_send_interval <= 0) {
		http.Error(w, "values must be greater than 0", 400)
		return
	}

	settings = in_settings

	err := persistSettings()
	if err != nil {
		http.Error(w, "trouble persisting settings, create bucket", 500)
		return
	}

	w.Write([]byte("ok"))
}

//websocket stuff
func reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func writer(ws *websocket.Conn) {
	pingTicker := time.NewTicker(pingPeriod)
	beaconTicker := time.NewTicker(beaconPeriod)
	defer func() {
		pingTicker.Stop()
		beaconTicker.Stop()
		ws.Close()
	}()
	for {
		select {
		case <-beaconTicker.C:

			http_results_lock.RLock()
			js, err := json.Marshal(http_results)
			http_results_lock.RUnlock()

			if err != nil {
				js = []byte("error")
			}

			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.TextMessage, js); err != nil {
				return
			}
		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	go writer(ws)
	reader(ws)
}

func latestBeaconWriter(ws *websocket.Conn) {
	pingTicker := time.NewTicker(pingPeriod)
	beaconTicker := time.NewTicker(beaconPeriod)
	defer func() {
		pingTicker.Stop()
		beaconTicker.Stop()
		ws.Close()
	}()
	for {
		select {
		case <-beaconTicker.C:

			latest_list_lock.RLock()
			var la = make([]Beacon, 0)
			for _, b := range Latest_beacons_list {
				la = append(la, b)
			}
			latest_list_lock.RUnlock()
			js, err := json.Marshal(la)

			if err != nil {
				js = []byte("error")
			}

			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.TextMessage, js); err != nil {
				return
			}
		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func serveLatestBeaconsWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	go latestBeaconWriter(ws)
	reader(ws)
}
