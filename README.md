# Weather Station InfluxDB Logger

A lightweight, high-performance Go application that acts as a bridge between Personal Weather Stations (PWS) and InfluxDB v2. It listens for HTTP GET requests (standard PWS upload format), parses the query parameters, calculates derived metrics (like Dew Point), and writes the data to an InfluxDB bucket.

## Features

* **Fast & Efficient:** Refactored to remove heavy reflection; uses direct type parsing.
* **Automatic Calculations:** Calculates **Dew Point** (Indoor and Outdoor) based on temperature and humidity.
* **InfluxDB v2 Support:** Native support for InfluxDB 2.x using the official Go client.
* **Container Ready:** Includes Dockerfile and instructions for containerized deployment.
* **Structured Logging:** Uses Go's `slog` for clear, standard log output.

## Quick Start (Docker)

The easiest way to run this is using the pre-built Docker image.

### Option 1: Docker CLI

**Note:** Replace `192.168.1.x` with the actual IP address of your InfluxDB server. Do not use `localhost` inside the container.

```bash
docker run -d \
  --name weather-receiver \
  -p 8080:8080 \
  -e INFLUXDB_URL="http://192.168.1.x:8086" \
  -e INFLUXDB_ORG="my-org" \
  -e INFLUXDB_BUCKET="weather" \
  -e INFLUXDB_TOKEN="your-token-here" \
  slydog/weather-receiver:latest
```

### Option 2: Docker Compose

Create a `docker-compose.yml`:

```yaml
services:
  weather-receiver:
    image: slydog/weather-receiver:latest
    container_name: weather-receiver
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      # CRITICAL: Do not use "localhost". Use the actual IP of the InfluxDB server.
      - INFLUXDB_URL=http://192.168.1.50:8086
      - INFLUXDB_ORG=my-org
      - INFLUXDB_BUCKET=weather
      - INFLUXDB_TOKEN=my-super-secret-token
```

Then run:

```bash
docker-compose up -d
```

## Setup Your Weather Station

Configure your weather station (Ambient Weather, Ecowitt, etc.) to use the **"Custom Server"** settings:

* **Server/Hostname:** The IP address of the machine running this container.
* **Port:** `8080`
* **Path:** `/` (Root)
* **Station ID/Key:** (Optional, will be logged as the `passkey` tag).

## Configuration Reference

The application is configured primarily via Environment Variables.

| Variable | Description | Example |
| :--- | :--- | :--- |
| `INFLUXDB_URL` | The URL of your InfluxDB server | `http://192.168.1.50:8086` |
| `INFLUXDB_TOKEN` | Your InfluxDB API Token | `your-super-secret-token` |
| `INFLUXDB_ORG` | Your InfluxDB Organization name | `my-org` |
| `INFLUXDB_BUCKET` | The bucket to write data to | `weather` |

### Command Line Flags (For running from source)

| Flag | Default | Description |
| :--- | :--- | :--- |
| `-port` | `8080` | The HTTP port the server listens on |

## Building from Source (Development)

If you want to modify the code or build the container locally:

**Prerequisites:**
* Go 1.24+
* Docker

**Build & Run with Compose:**
This repository includes a `docker-compose.yml` configured to build from source.

1. Edit `docker-compose.yml` to set your environment variables.
2. Run:
   ```bash
   docker-compose up -d --build
   ```