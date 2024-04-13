package main

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/mcuadros/go-lookup"
)

type WeatherData struct {
	Passkey        string  `json:"passkey"`
	StationType    string  `json:"stationtype"`
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
}

func main() {
	// InfluxDB connection details (replace with your credentials)

	// client := influxdb.NewClient(influxURL, token)
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     "http://192.168.86.36:8086",
		Username: "mydb",
		Password: "mydb",
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
		// Parse the query string
		queryParams := r.URL.Query()

		// Create a new WeatherData struct
		data := WeatherData{}

		// Loop through query params and populate struct fields
		for key, value := range queryParams {

			switch key {
			case "PASSKEY":
				data.Passkey = value[0]

			case "stationtype":
				data.StationType = value[0]

			case "dateutc":
				data.DateUtc = strings.Replace(value[0], " ", "T", 1) + "Z"

			// Parse numeric values
			case "tempf", "windspeedmph", "windgustmph", "maxdailygust", "solarradiation", "hourlyrainin", "eventrainin", "dailyrainin", "weeklyrainin", "monthlyrainin", "yearlyrainin", "baromrelin", "baromabsin":
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

		// Convert struct to a point
		tags := map[string]string{
			"passkey":     data.Passkey,
			"stationtype": data.StationType,
		}
		fields := map[string]interface{}{
			"passkey":        data.Passkey,
			"stationtype":    data.StationType,
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
			return
		}
		fmt.Fprintf(w, "Data posted to InfluxDB successfully!")
		fmt.Println(time.Now(), " - ", tags, fields)
	})

	// Start the server
	fmt.Println("Server listening on port 8080")
	if http.ListenAndServe(":8080", router) != nil {
		fmt.Println("ListenAndServe failed")
	}
}
