#!/bin/sh

# modify the mqtt username, password, and host_path for your setup, host path includes the host and port 

while true
do
	./presence -mqtt_username=mqtt_user -mqtt_password=mqtt_password -mqtt_host=localhost:1883
	sleep 1
done
