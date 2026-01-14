# Weather Station InfluxDB Logger

A lightweight, high-performance Go application that acts as a bridge between Personal Weather Stations (PWS) and InfluxDB v2. It listens for HTTP GET requests (standard PWS upload format), parses the query parameters, calculates derived metrics (like Dew Point), and writes the data to an InfluxDB bucket.

## Features

* **Fast & Efficient:** Refactored to remove heavy reflection; uses direct type parsing.
* **Automatic Calculations:** Calculates **Dew Point** (Indoor and Outdoor) based on temperature and humidity.
* **InfluxDB v2 Support:** Native support for InfluxDB 2.x using the official Go client.
* **Container Ready:** Includes Dockerfile and instructions for containerized deployment.
* **Structured Logging:** Uses Go's `slog` for clear, standard log output.

## Prerequisites

* **InfluxDB v2 Server** (Running locally or remotely)
* **Go 1.24+** (Only if building from source)
* **Docker** (Optional, for containerized running)

## Configuration

The application is configured primarily via Environment Variables.

| Variable | Description | Example |
| :--- | :--- | :--- |
| `INFLUXDB_URL` | The URL of your InfluxDB server | `http://192.168.1.50:8086` |
| `INFLUXDB_TOKEN` | Your InfluxDB API Token | `your-super-secret-token` |
| `INFLUXDB_ORG` | Your InfluxDB Organization name | `my-org` |
| `INFLUXDB_BUCKET` | The bucket to write data to | `weather` |

### Command Line Flags

| Flag | Default | Description |
| :--- | :--- | :--- |
| `-port` | `8080` | The HTTP port the server listens on |

## Deployment

### Option 1: Docker (Recommended)

Since you have an existing InfluxDB server, use this command to launch the app.

**Note:** Replace `192.168.1.x` with the actual IP address of your InfluxDB server. Do not use `localhost` inside the container.

```bash
docker run -d \
  --name slydog/weather-receiver \
  -p 8080:8080 \
  -e INFLUXDB_URL="[http://192.168.1.](http://192.168.1.)x:8086" \
  -e INFLUXDB_ORG="my-org" \
  -e INFLUXDB_BUCKET