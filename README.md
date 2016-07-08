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

#### 0. Compile it
0. Go to releases and download the executable for your environment
or
1. Clone this repo and compile the program with "go get" then "go build"


#### 1. Run it
1. Run "./presence -h" for a list of options. But basically you should specify the MQTT server and username/password to connect to. This MQTT server should be the same one your Happy Bubbles detectors publish to. The presence server will subscribe to all their topics and being tracking location.
2. From the web interface (runs on port 3000 by default) you can visit the "Latest Seen Beacons" tab in the top right and find your beacon in there, then click "Add this beacon" which will let you give it a friendly name.
3. If you subscribe to the MQTT topic "happy-bubbles/presence/changes" you will get JSON messages that tell you when your added beacons change location, disappear entirely, or appear at any new location.

#### TODO
* Publish the HTTP and MQTT API for presence server
