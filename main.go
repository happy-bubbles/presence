package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/boltdb/bolt"

	"github.com/yosssi/gmq/mqtt"
	"github.com/yosssi/gmq/mqtt/client"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"./kalmanfilter"
)

const (
	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	beaconPeriod = 2 * time.Second
)

// data structures

type Settings struct {
	Location_confidence    int64 `json:"location_confidence"`
	Last_seen_threshold    int64 `json:"last_seen_threshold"`
	Last_reading_threshold int64 `json:"last_reading_threshold"`
}

type Incoming_json struct {
	Hostname         string `json:"hostname"`
	MAC              string `json:"mac"`
	RSSI             int64  `json:"rssi"`
	Is_scan_response string `json:"is_scan_response"`
	Ttype            string `json:"type"`
	Data             string `json:"data"`
	Beacon_type      string `json:"beacon_type"`
	UUID             string `json:"uuid"`
	Major            string `json:"major"`
	Minor            string `json:"minor"`
	TX_power         string `json:"tx_power"`
	Namespace        string `json:"namespace"`
	Instance_id      string `json:"instance_id"`
}

type Advertisement struct {
	ttype   string
	content string
	seen    int64
}

type Found_beacon struct {
	beacon_id    string
	last_seen    int64
	filter_data  *kalmanfilter.FilterData
	latestKalman float64
}

type Location struct {
	name          string
	found_beacons map[string]Found_beacon
	lock          sync.RWMutex
}

type Best_location struct {
	distance  float64
	name      string
	last_seen int64
}

type HTTP_location struct {
	Distance    float64 `json:"distance"`
	Name        string  `json:"name"`
	Beacon_name string  `json:"beacon_name"`
	Beacon_id   string  `json:"beacon_id"`
	Location    string  `json:"location"`
	Last_seen   int64   `json:"last_seen"`
}

type Location_change struct {
	Beacon_ref        Beacon `json:"beacon_info"`
	Name              string `json:"name"`
	Beacon_name       string `json:"beacon_name"`
	Previous_location string `json:"previous_location"`
	New_location      string `json:"new_location"`
	Timestamp         int64  `json:"timestamp"`
}

type HA_message struct {
	Beacon_id   string  `json:"id"`
	Beacon_name string  `json:"name"`
	Distance    float64 `json:"distance"`
}

type HTTP_locations_list struct {
	Results []HTTP_location `json:"results"`
}

type Beacon struct {
	Name                        string        `json:"name"`
	Beacon_id                   string        `json:"beacon_id"`
	Beacon_type                 string        `json:"beacon_type"`
	Beacon_location             string        `json:"beacon_location"`
	Last_seen                   int64         `json:"last_seen"`
	Incoming_JSON               Incoming_json `json:"incoming_json"`
	Distance                    float64       `json:"distance"`
	Previous_location           string
	Previous_confident_location string
	Location_confidence         int64
}

type Beacons_list struct {
	Beacons map[string]Beacon `json:"beacons"`
	lock    sync.RWMutex
}

// GLOBALS

var BEACONS Beacons_list

var locations map[string]Location
var cli *client.Client

var http_results HTTP_locations_list
var http_results_lock sync.RWMutex

var Latest_beacons_list map[string]Beacon
var latest_list_lock sync.RWMutex

var db *bolt.DB
var err error

var world = []byte("presence")

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var settings = Settings{
	Location_confidence:    8,
	Last_seen_threshold:    45,
	Last_reading_threshold: 8,
}

// utility function

func twos_comp(inp string) int64 {
	i, _ := strconv.ParseInt("0x"+inp, 0, 64)
	return i - 256
}

func getBeaconID(incoming Incoming_json) string {
	unique_id := fmt.Sprintf("%s", incoming.MAC)
	if incoming.Beacon_type == "ibeacon" {
		unique_id = fmt.Sprintf("%s_%s_%s", incoming.UUID, incoming.Major, incoming.Minor)
	} else if incoming.Beacon_type == "eddystone" {
		unique_id = fmt.Sprintf("%s_%s", incoming.Namespace, incoming.Instance_id)
	}
	return unique_id
}

func getiBeaconDistance(rssi int64, power string) float64 {
	ratio := float64(rssi) * (1.0 / float64(twos_comp(power)))
	distance := 100.0
	if ratio < 1.0 {
		distance = math.Pow(ratio, 10)
	} else {
		distance = (0.89976)*math.Pow(ratio, 7.7095) + 0.111
	}
	return distance
}

func getBeaconDistance(incoming Incoming_json) float64 {
	distance := 1000.0
	if incoming.Beacon_type == "ibeacon" {
		distance = getiBeaconDistance(incoming.RSSI, incoming.TX_power)
	} else if incoming.Beacon_type == "eddystone" {
		//TODO: fix this, probably not the way to do this calc with eddystone
		distance = getiBeaconDistance(incoming.RSSI, incoming.TX_power)
	} else {
		//return the absolute value of RSSI. this is fine since should always be below 0 and the closer to 0 the better, so smaller is better just like ibeacon distance
		distance = math.Abs(float64(incoming.RSSI))
	}
	return distance
}

func sendHARoomMessage(beacon_id string, beacon_name string, distance float64, location string, cl *client.Client) {
	//first make the json
	ha_msg, err := json.Marshal(HA_message{Beacon_id: beacon_id, Beacon_name: beacon_name, Distance: distance})
	if err != nil {
		panic(err)
	}

	//send the message to HA
	err = cl.Publish(&client.PublishOptions{
		QoS:       mqtt.QoS1,
		TopicName: []byte("happy-bubbles/presence/ha/" + location),
		Message:   ha_msg,
	})
	if err != nil {
		panic(err)
	}
}

func getLikelyLocations(last_seen_threshold int64, last_reading_threshold int64, locations map[string]Location, cl *client.Client) {
	// create the http results structure
	http_results_lock.Lock()
	http_results = HTTP_locations_list{}
	http_results.Results = make([]HTTP_location, 0)
	http_results_lock.Unlock()

	should_persist := false

	// iterate through the beacons we want to search for
	for _, beacon := range BEACONS.Beacons {
		//fmt.Printf("doing iteration and saw %s with ID %s\n", beacon_name, beacon_id)
		//fmt.Printf("num locs %d\n", len(locations))
		best_location := Best_location{}
		//go through each location
		now := time.Now().Unix()
		for _, location := range locations {
			//fmt.Printf("doing iteration and saw location %s\n", location_name)
			// get last_seen for this location
			location.lock.RLock()
			found_b, ok := location.found_beacons[beacon.Beacon_id]
			if ok {
				//fmt.Printf("found %s in %s\n", beacon_id, location_name)
				ldistance := location.found_beacons[beacon.Beacon_id].latestKalman
				if (now - found_b.last_seen) > last_seen_threshold {
					//continue
					ldistance += 10.0
					fmt.Println("\n", location.name, " exceeded threshold, now distance is", ldistance, (now - found_b.last_seen), "\n")
				}

				if best_location == (Best_location{}) {
					best_location = Best_location{name: location.name, distance: ldistance, last_seen: location.found_beacons[beacon.Beacon_id].last_seen}
				} else if ldistance < best_location.distance {
					best_location = Best_location{name: location.name, distance: ldistance, last_seen: location.found_beacons[beacon.Beacon_id].last_seen}
				}
			}
			location.lock.RUnlock()
		}

		// debug stuff, show other candidates

		/*
			fmt.Printf("DEBUG: %s - best location: %s \n", beacon.Name, best_location.name)
			for _, location := range locations {
				avg_distance := location.found_beacons[beacon.Beacon_id].latestKalman
				now = time.Now().Unix()
				ago := now - location.found_beacons[beacon.Beacon_id].last_seen
				fmt.Printf("\t%s - average: %f, metrics: %d, last_seen: %d\n", location.name, avg_distance, len(location.found_beacons[beacon.Beacon_id].beacon_metrics), ago)
				fmt.Printf("\t\t")
				for _, met := range location.found_beacons[beacon.Beacon_id].beacon_metrics {
					fmt.Printf("%f ", met.distance)
				}
				fmt.Printf("\n")
			}
			fmt.Printf("\n")
		*/

		//filter, only let this location become best if it was X times in a row
		if best_location.name == beacon.Previous_location {
			beacon.Location_confidence = beacon.Location_confidence + 1
		} else {
			beacon.Location_confidence = 0
		}

		if beacon.Location_confidence == settings.Location_confidence && beacon.Previous_confident_location != best_location.name {
			// location has changed, send an mqtt message

			should_persist = true
			fmt.Printf("detected a change!!! %s - from %s to %s\n\n", beacon.Name, beacon.Previous_confident_location, best_location.name)

			// just for good measure, should have to earn it
			beacon.Location_confidence = 0

			//first make the json
			js, err := json.Marshal(Location_change{Beacon_ref: beacon, Name: beacon.Name, Beacon_name: beacon.Name, Previous_location: beacon.Previous_confident_location, New_location: best_location.name, Timestamp: time.Now().Unix()})
			if err != nil {
				continue
			}

			//send the message
			err = cl.Publish(&client.PublishOptions{
				QoS:       mqtt.QoS1,
				TopicName: []byte("happy-bubbles/presence/changes"),
				Message:   js,
			})
			if err != nil {
				panic(err)
			}

			beacon.Previous_confident_location = best_location.name

			// clear all previous entries of this beacon from all locations, except this best one
			//log.Println("before clear")
			for k, location := range locations {
				log.Println(location.name, beacon.Name)
				if location.name == best_location.name {
					continue
				}
				log.Println("deleting ", beacon.Name, "from ", location.name)
				delete(location.found_beacons, beacon.Name)
				/*
					locbeac := location.found_beacons[beacon.Name]
					locbeac.beacon_metrics = make([]Beacon_metric, 1)
					location.found_beacons[beacon.Name] = locbeac
				*/
				locations[k] = location
			}

			/*
				log.Println("after clear")
				for _, location := range locations {
					log.Println(location.name, beacon.Name, len(location.found_beacons[beacon.Name].beacon_metrics))
				}
			*/
		}

		//create an http result from this
		r := HTTP_location{}
		r.Distance = best_location.distance
		r.Name = beacon.Name
		r.Beacon_name = beacon.Name
		r.Beacon_id = beacon.Beacon_id
		r.Location = beacon.Previous_confident_location
		r.Last_seen = best_location.last_seen
		beacon.Previous_location = best_location.name

		BEACONS.lock.Lock()
		BEACONS.Beacons[beacon.Beacon_id] = beacon
		BEACONS.lock.Unlock()

		http_results_lock.Lock()
		http_results.Results = append(http_results.Results, r)
		http_results_lock.Unlock()

		if best_location.name != "" {
			sendHARoomMessage(beacon.Beacon_id, beacon.Name, best_location.distance, best_location.name, cl)
		}

		//fmt.Printf("\n\n%s is most likely in %s with average distance %f \n\n", beacon.Name, best_location.name, best_location.distance)
		// publish this to a topic
		// Publish a message.
		err := cl.Publish(&client.PublishOptions{
			QoS:       mqtt.QoS0,
			TopicName: []byte("happy-bubbles/presence"),
			Message:   []byte(fmt.Sprintf("%s is most likely in %s with average distance %f", beacon.Name, best_location.name, best_location.distance)),
		})
		if err != nil {
			panic(err)
		}
	}

	if should_persist {
		persistBeacons()
	}
}

func getLikelyLocationsPoller(locations map[string]Location, cl *client.Client) {
	for {
		<-time.After(1 * time.Second)
		go getLikelyLocations(settings.Last_seen_threshold, settings.Last_reading_threshold, locations, cl)
	}
}

var http_host_path_ptr *string

func main() {
	http_host_path_ptr = flag.String("http_host_path", "localhost:5555", "The host:port that the HTTP server should listen on")

	mqtt_host_ptr := flag.String("mqtt_host", "localhost:1883", "The host:port of the MQTT server to listen for Happy Bubbles beacons on")
	mqtt_username_ptr := flag.String("mqtt_username", "none", "The username needed to connect to the MQTT server, 'none' if it doesn't need one")
	mqtt_password_ptr := flag.String("mqtt_password", "none", "The password needed to connect to the MQTT server, 'none' if it doesn't need one")
	mqtt_client_id_ptr := flag.String("mqtt_client_id", "happy-bubbles-presence-detector", "The client ID for the MQTT server")

	flag.Parse()

	//open the database
	db, err = bolt.Open("presence.db", 0644, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// load initial BEACONS
	BEACONS.Beacons = make(map[string]Beacon)
	// retrieve the data

	// create bucket if not exist
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(world)
		if err != nil {
			return err
		}
		return nil
	})

	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(world)
		if bucket == nil {
			return err
		}

		key := []byte("beacons_list")
		val := bucket.Get(key)
		if val != nil {
			buf := bytes.NewBuffer(val)
			dec := gob.NewDecoder(buf)
			err = dec.Decode(&BEACONS)
			if err != nil {
				log.Fatal("decode error:", err)
			}
		}

		key = []byte("settings")
		val = bucket.Get(key)
		if val != nil {
			buf := bytes.NewBuffer(val)
			dec := gob.NewDecoder(buf)
			err = dec.Decode(&settings)
			if err != nil {
				log.Fatal("decode error:", err)
			}
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	//debug list them out
	/*
		for _, beacon := range BEACONS.Beacons {
			fmt.Println("Database has known beacon: " + beacon.Beacon_id + " " + beacon.Name)
		}
	*/
	//fmt.Println("Settings has %#v\n", settings)

	Latest_beacons_list = make(map[string]Beacon)

	//create a map of locations, looked up by hostnames
	locations := make(map[string]Location)

	// Set up channel on which to send signal notifications.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	// Create an MQTT Client.
	cli := client.New(&client.Options{
		// Define the processing of the error handler.
		ErrorHandler: func(err error) {
			fmt.Println(err)
		},
	})
	// Terminate the Client.
	defer cli.Terminate()

	// Connect to the MQTT Server.
	err = cli.Connect(&client.ConnectOptions{
		Network:  "tcp",
		Address:  *mqtt_host_ptr,
		ClientID: []byte(*mqtt_client_id_ptr),
		UserName: []byte(*mqtt_username_ptr),
		Password: []byte(*mqtt_password_ptr),
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(" _   _    _    ____  ______   __  ____  _   _ ____  ____  _     _____ ____\n| | | |  / \\  |  _ \\|  _ \\ \\ / / | __ )| | | | __ )| __ )| |   | ____/ ___|\n| |_| | / _ \\ | |_) | |_) \\ V /  |  _ \\| | | |  _ \\|  _ \\| |   |  _| \\___ \\\n|  _  |/ ___ \\|  __/|  __/ | |   | |_) | |_| | |_) | |_) | |___| |___ ___) |\n|_| |_/_/   \\_\\_|   |_|    |_|   |____/ \\___/|____/|____/|_____|_____|____/")
	fmt.Println("\n ")
	fmt.Println("CONNECTED TO MQTT")
	fmt.Println("\n ")
	fmt.Println("Visit http://" + *http_host_path_ptr + " on your browser to see the web interface")
	fmt.Println("\n ")

	// Subscribe to topics.
	err = cli.Subscribe(&client.SubscribeOptions{
		SubReqs: []*client.SubReq{
			&client.SubReq{
				//TopicFilter: []byte("happy-bubbles/ble/+/ibeacon/+"),
				TopicFilter: []byte("happy-bubbles/ble/#"),
				QoS:         mqtt.QoS0,
				// Define the processing of the message handler.
				Handler: func(topicName, message []byte) {
					incoming := Incoming_json{}
					json.Unmarshal(message, &incoming)

					this_beacon_id := getBeaconID(incoming)

					now := time.Now().Unix()

					//fmt.Println("saw " + this_beacon_id + " at " + incoming.Hostname)

					//if this beacon isn't in our search list, add it to the latest_beacons pile.
					BEACONS.lock.RLock()
					beacon, ok := BEACONS.Beacons[this_beacon_id]
					BEACONS.lock.RUnlock()
					if !ok {
						//should be unique
						//if it's already in list, forget it.
						latest_list_lock.Lock()
						x, ok := Latest_beacons_list[this_beacon_id]
						if ok {
							//update its timestamp
							x.Last_seen = now
							x.Incoming_JSON = incoming
							x.Distance = getBeaconDistance(incoming)
							Latest_beacons_list[this_beacon_id] = x
						} else {
							Latest_beacons_list[this_beacon_id] = Beacon{Beacon_id: this_beacon_id, Beacon_type: incoming.Beacon_type, Last_seen: now, Incoming_JSON: incoming, Beacon_location: incoming.Hostname, Distance: getBeaconDistance(incoming)}
						}
						for k, v := range Latest_beacons_list {
							if (now - v.Last_seen) > 10 { // 10 seconds
								delete(Latest_beacons_list, k)
							}
						}
						latest_list_lock.Unlock()
						return
					}

					BEACONS.lock.Lock()
					beacon.Incoming_JSON = incoming
					beacon.Last_seen = now
					beacon.Beacon_type = incoming.Beacon_type
					BEACONS.Beacons[beacon.Beacon_id] = beacon
					BEACONS.lock.Unlock()

					this_distance := getBeaconDistance(incoming)

					//lookup location by hostname in locations
					location, ok := locations[incoming.Hostname]
					if !ok {
						//create the location
						locations[incoming.Hostname] = Location{}
						location, ok = locations[incoming.Hostname]
						location.found_beacons = make(map[string]Found_beacon)
						//fmt.Println(location.name + " new location so making found_beacons map")
						location.name = incoming.Hostname
					}

					//now look for our beacon in founds
					location.lock.RLock()
					this_found, ok := location.found_beacons[this_beacon_id]
					location.lock.RUnlock()
					if !ok {
						//create kalman filter for this beacon
						this_found.filter_data = new(kalmanfilter.FilterData)
						fmt.Println(location, "first making new kalman")

						this_found.filter_data.R = 1
						this_found.filter_data.Q = 1
						this_found.filter_data.A = 1
						this_found.filter_data.B = 0
						this_found.filter_data.C = 1

					}
					if this_found.filter_data == nil {
						this_found.filter_data = new(kalmanfilter.FilterData)

						this_found.filter_data.R = 1
						this_found.filter_data.Q = 1
						this_found.filter_data.A = 1
						this_found.filter_data.B = 0
						this_found.filter_data.C = 1
						fmt.Println(location, "nil so making new kalman")
					}
					this_found.beacon_id = this_beacon_id
					this_found.last_seen = now
					// add the kalman filter to this found beacon
					//filval := this_found.filter_data.Update(this_metric.distance, 1.00, 10000000)
					filval := this_found.filter_data.Update(this_distance)
					this_found.latestKalman = filval
					fmt.Println("KALMAN", filval, " -- ", location.name, this_found.beacon_id, this_distance)

					location.lock.Lock()
					location.found_beacons[this_beacon_id] = this_found
					location.lock.Unlock()
					locations[incoming.Hostname] = location
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	// create a thread for finding all the closest beacons
	go getLikelyLocationsPoller(locations, cli)

	go startServer()

	// Wait for receiving a signal.
	<-sigc

	// Disconnect the Network Connection.
	if err := cli.Disconnect(); err != nil {
		panic(err)
	}
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
	js, err := json.Marshal(http_results.Results)
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

func beaconsEditHandler(w http.ResponseWriter, r *http.Request) {
	js, err := json.Marshal(http_results.Results)
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
		(in_settings.Last_reading_threshold <= 0) {
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
			js, err := json.Marshal(http_results.Results)
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
