package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// Config holds the application configuration
type Config struct {
	Port         int
	InfluxToken  string
	InfluxURL    string
	InfluxOrg    string
	InfluxBucket string
}

// WeatherData holds the parsed metrics
type WeatherData struct {
	Timestamp      time.Time
	Passkey        string
	TempF          float64
	Humidity       int
	WindSpeedMph   float64
	WindGustMph    float64
	MaxDailyGust   float64
	WindDir        int
	WindDir_Avg10m int
	Uv             int
	SolarRadiation float64
	HourlyRainIn   float64
	EventRainIn    float64
	DailyRainIn    float64
	WeeklyRainIn   float64
	MonthlyRainIn  float64
	YearlyRainIn   float64
	BattOut        int
	BattRain       int
	TempInF        float64
	HumidityIn     int
	BaromRelIn     float64
	BaromAbsIn     float64
	BattIn         int
	DewPt          float64 // Calculated
	DewPtIn        float64 // Calculated
}

// WeatherServer handles HTTP requests and InfluxDB writes
type WeatherServer struct {
	writeAPI api.WriteAPIBlocking
	logger   *slog.Logger
}

func main() {
	// 1. Configuration
	cfg := loadConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// 2. InfluxDB Setup
	client := influxdb2.NewClient(cfg.InfluxURL, cfg.InfluxToken)
	defer client.Close()

	// Ensure we can connect before starting
	// Note: Ping is not strictly necessary for NewClient, but good for fail-fast
	// implementation if desired.

	writeAPI := client.WriteAPIBlocking(cfg.InfluxOrg, cfg.InfluxBucket)

	server := &WeatherServer{
		writeAPI: writeAPI,
		logger:   logger,
	}

	// 3. HTTP Server Setup
	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleWeather)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	logger.Info("Starting server", "port", cfg.Port, "influx_url", cfg.InfluxURL)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

func loadConfig() Config {
	port := flag.Int("port", 8080, "Port to listen on")
	flag.Parse()

	return Config{
		Port:         *port,
		InfluxToken:  os.Getenv("INFLUXDB_TOKEN"),
		InfluxURL:    os.Getenv("INFLUXDB_URL"),
		InfluxOrg:    os.Getenv("INFLUXDB_ORG"),
		InfluxBucket: os.Getenv("INFLUXDB_BUCKET"),
	}
}

func (s *WeatherServer) handleWeather(w http.ResponseWriter, r *http.Request) {
	// Parse Query Parameters
	q := r.URL.Query()

	// Parse Data
	data, err := parseWeatherData(q)
	if err != nil {
		s.logger.Warn("Failed to parse weather data", "error", err)
		http.Error(w, fmt.Sprintf("Bad Request: %v", err), http.StatusBadRequest)
		return
	}

	// Calculate Dew Points
	data.DewPt = computeDewPt(data.TempF, data.Humidity)
	data.DewPtIn = computeDewPt(data.TempInF, data.HumidityIn)

	// Create Influx Point
	p := createInfluxPoint(data)

	// Write to InfluxDB
	if err := s.writeAPI.WritePoint(context.Background(), p); err != nil {
		s.logger.Error("Failed to write to InfluxDB", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	s.logger.Info("Data processed successfully", "timestamp", data.Timestamp, "passkey", data.Passkey)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Data posted to InfluxDB successfully!")
}

func parseWeatherData(q url.Values) (WeatherData, error) {
	var d WeatherData
	var err error

	// Helper to handle case-insensitivity if needed, though usually PWS sends exact keys.
	// We assume keys match the PWS standard (usually lowercase).

	d.Passkey = q.Get("PASSKEY") // Usually uppercase from some stations
	if d.Passkey == "" {
		d.Passkey = q.Get("passkey")
	}

	// Parse Date
	dateStr := q.Get("dateutc")
	if dateStr == "" {
		d.Timestamp = time.Now() // Fallback if missing
	} else {
		// Fix format: "2023-01-01 12:00:00" -> "2023-01-01T12:00:00Z"
		cleanDate := strings.Replace(dateStr, " ", "T", 1) + "Z"
		d.Timestamp, err = time.Parse(time.RFC3339, cleanDate)
		if err != nil {
			return d, fmt.Errorf("invalid date format: %v", err)
		}
	}

	// Floating Point Values
	d.TempF = getFloat(q, "tempf")
	d.WindSpeedMph = getFloat(q, "windspeedmph")
	d.WindGustMph = getFloat(q, "windgustmph")
	d.MaxDailyGust = getFloat(q, "maxdailygust")
	d.SolarRadiation = getFloat(q, "solarradiation")
	d.HourlyRainIn = getFloat(q, "hourlyrainin")
	d.EventRainIn = getFloat(q, "eventrainin")
	d.DailyRainIn = getFloat(q, "dailyrainin")
	d.WeeklyRainIn = getFloat(q, "weeklyrainin")
	d.MonthlyRainIn = getFloat(q, "monthlyrainin")
	d.YearlyRainIn = getFloat(q, "yearlyrainin")
	d.TempInF = getFloat(q, "tempinf")
	d.BaromRelIn = getFloat(q, "baromrelin")
	d.BaromAbsIn = getFloat(q, "baromabsin")

	// Integer Values
	d.Humidity = getInt(q, "humidity")
	d.WindDir = getInt(q, "winddir")
	d.WindDir_Avg10m = getInt(q, "winddir_avg10m")
	d.Uv = getInt(q, "uv")
	d.BattOut = getInt(q, "battout")
	d.BattRain = getInt(q, "battrain")
	d.HumidityIn = getInt(q, "humidityin")
	d.BattIn = getInt(q, "battin")

	return d, nil
}

func createInfluxPoint(d WeatherData) *write.Point {
	tags := map[string]string{
		"passkey": d.Passkey,
	}

	fields := map[string]interface{}{
		"tempf":          d.TempF,
		"humidity":       d.Humidity,
		"windspeedmph":   d.WindSpeedMph,
		"windgustmph":    d.WindGustMph,
		"maxdailygust":   d.MaxDailyGust,
		"winddir":        d.WindDir,
		"winddir_avg10m": d.WindDir_Avg10m,
		"uv":             d.Uv,
		"solarradiation": d.SolarRadiation,
		"hourlyrainin":   d.HourlyRainIn,
		"eventrainin":    d.EventRainIn,
		"dailyrainin":    d.DailyRainIn,
		"weeklyrainin":   d.WeeklyRainIn,
		"monthlyrainin":  d.MonthlyRainIn,
		"yearlyrainin":   d.YearlyRainIn,
		"battout":        d.BattOut,
		"battrain":       d.BattRain,
		"tempinf":        d.TempInF,
		"humidityin":     d.HumidityIn,
		"baromrelin":     d.BaromRelIn,
		"baromabsin":     d.BaromAbsIn,
		"battin":         d.BattIn,
		"dewpt":          d.DewPt,
		"dewptin":        d.DewPtIn,
	}

	return write.NewPoint("weather", tags, fields, d.Timestamp)
}

// Helpers

func computeDewPt(temp float64, humidity int) float64 {
	// DP = T - 9/25(100 - RH)
	// Guard against division issues or nonsense values if needed,
	// though this linear approximation is standard for simple stations.
	return temp - (0.36 * (100.0 - float64(humidity)))
}

func getFloat(q url.Values, key string) float64 {
	val := q.Get(key)
	if val == "" {
		return 0.0
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0.0 // Return 0 on error, or you could return an error if strictness is required
	}
	return f
}

func getInt(q url.Values, key string) int {
	val := q.Get(key)
	if val == "" {
		return 0
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return i
}
