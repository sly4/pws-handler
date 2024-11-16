# pws-handler

This is a small go program to map HTTP POSTs from an AmbientWeather weather station to InfluxDB.  These weather stations can push data on a schedule using AmbientWeather or Weatherground protocol types.  This Go muxer listens for the requests and then posts the data to an InfluxDB instance and is written to handle the AmbientWeather protocol type.

Having the weather data in InfuxDB allows one to use Grafana or other graphing tools to create custom displays.  The WS5000 is hard to read from across a room.  Being able to use a raspberry pi with a larger display is nice.

My weather station is a WS5000 running AMBWeatherPro_V5.1.7.

## How to use
`pws-handler -port=8080 -IdbAddr=http://xxx.xxx.xxx.xxx:port -IdbUser=someuser -IdbPass=somepass`

## NOTE:
Set max-body-size=0 in your influxd.conf file.
