![Happy Bubbles Logo](happy_bubbles_logo.png?raw=true)

# Happy Bubbles - Bluetooth Beacon Presence Detection Server

This Go program is a server that subscribes to MQTT topics that the "Happy Bubbles" [Bluetooth Beacon Presence Detectors](https://www.happybubbles.tech/shop/) publish to. It checks to see which of the detectors found the strongest signal for a particular beacon, and then lets you access that info either over an API, or a web interface on http://localhost:3000/ but you can change that if you want.

It is designed to be used as a home-automation presence detection system. If you install the detectors through-out a home and family members carry beacons around the house, you can program your home automation hubs to take certain actions depending on who entered or left certain rooms. This server also publishes changes in location to a particular topic. So you can program your hub to listen for these and make the desired changes as they happen, to not have to keep polling it.

The detectors can be purchased from the Happy Bubbles shop at https://www.happybubbles.tech/shop/

There is more information about how the presence detection system works at https://www.happybubbles.tech/presence/ and how the detectors themselves work at https://www.happybubbles.tech/presence/detector

Here's some images of what it looks like

![Presence Added Screenshot](screenshot_added_beacons.png?raw=true)

![Presence All Found Screenshot](screenshot_latest_beacons.png?raw=true)

## How to set it up and use it

### Download, install, and run

The quickest and easiest way to get started is to download the latest release of the server from https://github.com/happy-bubbles/presence/releases and install it following the instructions. 

There is also a tutorial for installing it to a Raspberry Pi here: https://www.happybubbles.tech/presence/docs/rpi3_setup/

But if you want to do development on the system or install from source, take a look at the instructions below.

## Compile and install from source

#### 0. Compile it
0. Go to releases and download the executable for your environment
or
1. Clone this repo and compile the program with "go get" then "go build"


#### 1. Run it
1. Run "./presence -h" for a list of options. But basically you should specify the MQTT server and username/password to connect to. This MQTT server should be the same one your Happy Bubbles detectors publish to. The presence server will subscribe to all their topics and being tracking location.
2. From the web interface (runs on port 3000 by default) you can visit the "Latest Seen Beacons" tab in the top right and find your beacon in there, then click "Add this beacon" which will let you give it a friendly name.
3. If you subscribe to the MQTT topic "happy-bubbles/presence/changes" you will get JSON messages that tell you when your added beacons change location, disappear entirely, or appear at any new location.

#### MQTT

* The changes in location (when a beacon changes location or goes offline/online) are pushed to the "happy-bubbles/presence/changes" topic as they occur
* The messages for the presence server status are all sent out over MQTT as well. The current status of all added beacons will be pushed to the "happy-bubbles/presence" topic once a second.
The changes (when a beacon changes location or goes offline/online) by subscribing to "happy-bubbles/presence/changes"

The payloads of both messages are in JSON format.

## Run in a Docker container

#### 0. Generate and start the docker container
1. Start docker/build.sh
2. Start docker/run.sh
3. The interface is available at http://locahost:5555

#### 1. Settings
MQTT and web interface settings can be added/changed in docker/docker-entrypoint.sh

Example:
```./presence -http_host_path=0.0.0.0:5555 -mqtt_client_id=happy_buble -mqtt_host=localhost:1883 -mqtt_password=1234 -mqtt_username=homeassistant```

#### TODO
* Publish the HTTP and MQTT API for presence server
