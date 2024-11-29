package main

import (
	"flag"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/mcuadros/go-lookup"
)

type WeatherData struct {
	Passkey string `json:"passkey"`
	//StationType    string  `json:"stationtype"`
	DateUtc        string  `json:"dateutc"`
	TempF          float64 `json:"tempf"`
	Humidity       int     `json:"humidity"`
	WindSpeedMph   float64 `json:"windspeedmph"`
	WindGustMph    float64 `json:"windgustmph"`
	MaxDailyGust   float64 `json:"maxdailygust"`
	WindDir        int     `json:"winddir"`
	WindDir_Avg10m int     `json:"winddir_avg10m"`
	Uv             int     `json:"uv"`
	SolarRadiation float64 `json:"solarradiation"`
	HourlyRainIn   float64 `json:"hourlyrainin"`
	EventRainIn    float64 `json:"eventrainin"`
	DailyRainIn    float64 `json:"dailyrainin"`
	WeeklyRainIn   float64 `json:"weeklyrainin"`
	MonthlyRainIn  float64 `json:"monthlyrainin"`
	YearlyRainIn   float64 `json:"yearlyrainin"`
	BattOut        int     `json:"battout"`
	BattRain       int     `json:"battrain"`
	TempInF        float64 `json:"tempinf"`
	HumidityIn     int     `json:"humidityin"`
	BaromRelIn     float64 `json:"baromrelin"`
	BaromAbsIn     float64 `json:"baromabsin"`
	BattIn         int     `json:"battin"`
	DewPt          float64 `json:"dewpt"`
	DewPtIn        float64 `json:"dewptin"`
}

type Wunderground struct {
	Action             string  `json:"action"`           // [action=updateraw] -- always supply this parameter to indicate you are making a weather observation upload
	Id                 string  `json:"ID"`               // [ID as registered by wunderground.com]
	Password           string  `json:"PASSWORD"`         // [Station Key registered with this PWS ID, case sensitive]
	DateUtc            string  `json:"dateutc"`          // - [YYYY-MM-DD HH:MM:SS (mysql format)] In Universal Coordinated Time (UTC) Not local time
	WindDir            int     `json:"winddir"`          // - [0-360 instantaneous wind direction]
	WindSpeedMph       float64 `json:"windspeedmph"`     // - [mph instantaneous wind speed]
	WindGustMph        float64 `json:"windgustmph"`      // - [mph current wind gust, using software specific time period]
	WindGustDir        int     `json:"windgustdir"`      // - [0-360 using software specific time period]
	WindSpeedMph_avg2m float64 `json:"windspdmph_avg2m"` // - [mph 2 minute average wind speed mph]
	WindDir_avg2m      int     `json:"winddir_avg2m"`    // - [0-360 2 minute average wind direction]
	WindGustMph_10m    float64 `json:"windgustmph_10m"`  // - [mph past 10 minutes wind gust mph ]
	WindGustDir_10m    int     `json:"windgustdir_10m"`  // - [0-360 past 10 minutes wind gust direction]
	Humidity           int     `json:"humidity"`         // - [% outdoor humidity 0-100%]
	DewPtF             float64 `json:"dewptf"`           // - [F outdoor dewpoint F]
	TempF              float64 `json:"tempf"`            // - [F outdoor temperature]
	//   * for extra outdoor sensors use temp2f, temp3f, and so on
	HourlyRainIn float64 `json:"rainin"`      // - [rain inches over the past hour)] -- the accumulated rainfall in the past 60 min
	DailyRainIn  float64 `json:"dailyrainin"` // - [rain inches so far today in local time]
	BaromAbsIn   float64 `json:"baromin"`     // - [barometric pressure inches]
	//	Weather 			string	`json:"weather"`			// - [text] -- metar style (+RA)
	//clouds - [text] -- SKC, FEW, SCT, BKN, OVC
	//soiltempf - [F soil temperature]
	//* for sensors 2,3,4 use soiltemp2f, soiltemp3f, and soiltemp4f
	//soilmoisture - [%]
	//* for sensors 2,3,4 use soilmoisture2, soilmoisture3, and soilmoisture4
	//leafwetness  - [%]
	//+ for sensor 2 use leafwetness2
	SolarRadiation float64 `json:"solarradiation"` // - [W/m^2]
	Uv             int     `json:"UV"`             // - [index]
	//visibility - [nm visibility]
	TempInF    float64 `json:"indoortempf"`    // - [F indoor temperature F]
	HumidityIn int     `json:"indoorhumidity"` // - [% indoor humidity 0-100]
}

func computeDewPt(temp float64, humidity int) float64 {
	// DP = T - 9/25(100 - RH)
	dp := temp - (float64(9) / float64(25) * (100 - float64(humidity)))

	return dp
}

func hasProperty(s interface{}, propertyName string) bool {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return false
	}
	return v.FieldByName(propertyName).IsValid()
}

func main() {
	ListenPortPtr := flag.Int("port", 8080, "Port to listen on for requests from PWS")
	//PwsProtocol := flag.String("proto", "AmbientWeather", "AmbientWeather | Wunderground")

	// InfluxDB connection details (replace with your credentials)
	IdbAddrPtr := flag.String("IdbAddr", "http://192.168.86.36:8086", "URL to InfluxDB")
	IdbUserPtr := flag.String("IdbUser", "mydb", "InfluxDB username")
	IdbPassPtr := flag.String("IdbPass", "mydb", "InfluxDB password")

	flag.Parse()

	// client := influxdb.NewClient(influxURL, token)
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     *IdbAddrPtr,
		Username: *IdbUserPtr,
		Password: *IdbPassPtr,
	})
	if err != nil {
		fmt.Println("Error creating InfluxDB client:", err)
		return
	}
	defer c.Close()

	// Create a new point batch
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  "mydb",
		Precision: "s",
	})
	if err != nil {
		fmt.Println("Error creating point batch:", err)
		return
	}

	// Create a new router
	router := mux.NewRouter()

	// Handler for weather data endpoint
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//fmt.Println(r.URL)

		// Parse the query string
		queryParams := r.URL.Query()

		// Create a new WeatherData struct
		//var data interface{}
		//if *PwsProtocol == "Wunderground" {
		//	data = Wunderground{}
		//} else {
		data := WeatherData{}
		//}

		// Loop through query params and populate struct fields
		for key, value := range queryParams {

			switch key {
			case "PASSKEY":
				data.Passkey = value[0]

			//case "stationtype":
			//	data.StationType = value[0]

			case "dateutc":
				data.DateUtc = strings.Replace(value[0], " ", "T", 1) + "Z"

			// Parse numeric values
			case "tempf",
				"tempinf",
				"windspeedmph",
				"windgustmph",
				"maxdailygust",
				"solarradiation",
				"hourlyrainin",
				"eventrainin",
				"dailyrainin",
				"weeklyrainin",
				"monthlyrainin",
				"yearlyrainin",
				"baromrelin",
				"baromabsin":
				var fval float64
				if _, err := fmt.Sscanln(value[0], &fval); err != nil {
					fmt.Println("Error parsing value for", key, err)
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, "Error parsing value for %s", key)
					return
				}

				fieldValue, _ := lookup.LookupI(&data, key)
				fieldValue.Set(reflect.ValueOf(fval))

			// Parse integer values
			case "humidity",
				"winddir",
				"winddir_avg10m",
				"uv",
				"battout",
				"battrain",
				"humidityin",
				"battin":
				var ival int
				if _, err := fmt.Sscanln(value[0], &ival); err != nil {
					fmt.Println("Error parsing value for", key, err)
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, "Error parsing value for %s", key)
					return
				}
				fieldValue, _ := lookup.LookupI(&data, key)
				fieldValue.Set(reflect.ValueOf(ival))

			}
		}

		data.DewPt = computeDewPt(data.TempF, data.Humidity)
		data.DewPtIn = computeDewPt(data.TempInF, data.HumidityIn)

		// Convert struct to a point
		tags := map[string]string{
			"passkey": data.Passkey,
			//"stationtype": data.StationType,
		}
		fields := map[string]interface{}{
			//"passkey": data.Passkey,
			//"stationtype":    data.StationType,
			"tempf":          data.TempF,
			"humidity":       data.Humidity,
			"windspeedmph":   data.WindSpeedMph,
			"windgustmph":    data.WindGustMph,
			"maxdailygust":   data.MaxDailyGust,
			"winddir":        data.WindDir,
			"winddir_avg10m": data.WindDir_Avg10m,
			"uv":             data.Uv,
			"solarradiation": data.SolarRadiation,
			"hourlyrainin":   data.HourlyRainIn,
			"eventrainin":    data.EventRainIn,
			"dailyrainin":    data.DailyRainIn,
			"weeklyrainin":   data.WeeklyRainIn,
			"monthlyrainin":  data.MonthlyRainIn,
			"yearlyrainin":   data.YearlyRainIn,
			"battout":        data.BattOut,
			"battrain":       data.BattRain,
			"tempinf":        data.TempInF,
			"humidityin":     data.HumidityIn,
			"baromrelin":     data.BaromRelIn,
			"baromabsin":     data.BaromAbsIn,
			"battin":         data.BattIn,
			"dewpt":          data.DewPt,
			"dewptin":        data.DewPtIn,
		}

		pdate, err := time.Parse(time.RFC3339, data.DateUtc)
		if err != nil {
			fmt.Println("Error converting time:", err)
			return
		}

		p, err := client.NewPoint("weather", tags, fields, pdate)
		if err != nil {
			fmt.Println("Error creating NewPoint", err)
			return
		}

		bp.AddPoint(p)

		// Write the point to InfluxDB
		if err := c.Write(bp); err != nil {
			fmt.Println("Error writing batch points:", err)
			fmt.Println(tags)
			fmt.Println(fields)
			return
		}
		fmt.Fprintf(w, "Data posted to InfluxDB successfully!")
		fmt.Println(time.Now(), " - ", tags, fields)
	})

	// Start the server
	fmt.Printf("Server listening on port %d\n", *ListenPortPtr)
	fmt.Printf("Muxing to %s\n", *IdbAddrPtr)

	// Create the HTTP server
	srv := http.Server{
		Addr: ":" + strconv.Itoa(*ListenPortPtr),
		// ReadHeaderTimeout is the amount of time allowed to read
		// request headers. The connection's read deadline is reset
		// after reading the headers and the Handler can decide what
		// is considered too slow for the body. If ReadHeaderTimeout
		// is zero, the value of ReadTimeout is used. If both are
		// zero, there is no timeout.
		ReadHeaderTimeout: 15 * time.Second,

		// ReadTimeout is the maximum duration for reading the entire
		// request, including the body. A zero or negative value means
		// there will be no timeout.
		//
		// Because ReadTimeout does not let Handlers make per-request
		// decisions on each request body's acceptable deadline or
		// upload rate, most users will prefer to use
		// ReadHeaderTimeout. It is valid to use them both.
		ReadTimeout: 15 * time.Second,

		// WriteTimeout is the maximum duration before timing out
		// writes of the response. It is reset whenever a new
		// request's header is read. Like ReadTimeout, it does not
		// let Handlers make decisions on a per-request basis.
		// A zero or negative value means there will be no timeout.
		WriteTimeout: 10 * time.Second,

		// IdleTimeout is the maximum amount of time to wait for the
		// next request when keep-alives are enabled. If IdleTimeout
		// is zero, the value of ReadTimeout is used. If both are
		// zero, there is no timeout.
		IdleTimeout: 30 * time.Second,

		Handler: router,
	}

	// For per request timeouts applications can wrap all `http.HandlerFunc(...)` in
	// `http.TimeoutHandler`` and specify a timeout, but note the TimeoutHandler does not
	// start ticking until all headers have been read.

	// Listen with our custom server with timeouts configured
	//if srv.ListenAndServe() != nil {
	//	fmt.Println("ListenAndServe failed")
	//}

	if srv.ListenAndServe() != nil {
		fmt.Println("ListenAndServe failed")
	}
}
