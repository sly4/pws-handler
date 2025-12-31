package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"os"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"

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

func computeDewPt(temp float64, humidity int) float64 {
	// DP = T - 9/25(100 - RH)
	dp := temp - (float64(9) / float64(25) * (100 - float64(humidity)))

	return dp
}

func main() {
	ListenPortPtr := flag.Int("port", 8080, "Port to listen on for requests from PWS")

	// InfluxDB connection details (replace with your credentials)
	IdbToken := os.Getenv("INFLUXDB_TOKEN")
	IdbUrl := os.Getenv("INFLUXDB_URL")
	IdbOrg := os.Getenv("INFLUXDB_ORG")
	IdbBucket := os.Getenv("INFLUXDB_BUCKET")

	flag.Parse()

	IdbClient := influxdb2.NewClient(IdbUrl, IdbToken)

	// Create a new point batch
	IdbWriteAPI := IdbClient.WriteAPIBlocking(IdbOrg, IdbBucket)

	// Create a new router
	router := mux.NewRouter()

	// Handler for weather data endpoint
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//fmt.Println(r.URL)

		// Parse the query string
		queryParams := r.URL.Query()

		// Create a new WeatherData struct
		data := WeatherData{}

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
			case "tempf", "tempinf", "windspeedmph", "windgustmph", "maxdailygust", "solarradiation", "hourlyrainin", "eventrainin", "dailyrainin", "weeklyrainin", "monthlyrainin", "yearlyrainin", "baromrelin", "baromabsin":
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
			case "humidity", "winddir", "winddir_avg10m", "uv", "battout", "battrain", "humidityin", "battin":
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

		p := write.NewPoint("weather", tags, fields, pdate)

		// Write the point to InfluxDB
		if err := IdbWriteAPI.WritePoint(context.Background(), p); err != nil {
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
	fmt.Printf("Muxing to %s\n", *&IdbUrl)

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
