# MQTT Microservice

A professional MQTT microservice written in Go that serves as a client for a Laravel IoT Cloud backend. 
This microservice handles all MQTT communication (subscribe, publish, reconnects, handling SSL certificates), 
offloading the MQTT layer from the Laravel app.

## Features

- Supports multiple MQTT brokers simultaneously (HiveMQ Cloud, Mosquitto, etc.)
- Reads connection settings from environment variables
- SSL/TLS support with certificate validation
- Auto-reconnect logic and timeouts
- Simple HTTP API for Laravel to interact with MQTT
- Health check endpoint
- Comprehensive logging

## Architecture

The microservice is built with a clean, modular architecture:

- **Configuration Module**: Loads and validates connection settings from environment variables
- **MQTT Client Manager**: Manages connections to multiple MQTT brokers
- **HTTP API Server**: Exposes endpoints for publishing messages, managing subscriptions, and checking status
- **Logging Utility**: Provides consistent logging throughout the application

## API Endpoints

### Core MQTT Endpoints
- `POST /publish`: Publish a message to a topic
- `POST /subscribe`: Subscribe to a topic
- `POST /unsubscribe`: Unsubscribe from a topic
- `GET /status`: Get the status of all MQTT connections
- `GET /healthz`: Health check endpoint

### Monitoring Endpoints
- `GET /metrics`: Get metrics about the MQTT microservice
- `GET /logs`: View logs

### Database Endpoints
- `GET /messages`: Get messages from the database
- `GET /messages/{id}`: Get a specific message by ID
- `POST /messages/{id}/confirm`: Confirm a message
- `DELETE /messages/{id}`: Delete a specific message
- `DELETE /messages/confirmed`: Delete all confirmed messages

## Environment Variables

The microservice is configured using environment variables. Example:

```
MQTT_DEFAULT_CONNECTION=hivemq

# HiveMQ Cloud connection settings
MQTT_HIVEMQ_HOST=hfghgfhgfhgfhgf
MQTT_HIVEMQ_PORT=8883
MQTT_HIVEMQ_CLIENT_ID=laravel-backend
MQTT_HIVEMQ_CLEAN_SESSION=true
MQTT_HIVEMQ_ENABLE_LOGGING=true
MQTT_HIVEMQ_LOG_CHANNEL=stack
MQTT_TLS_ENABLED=true
MQTT_TLS_VERIFY_PEER=true
MQTT_TLS_CA_FILE=certificates/www-hivemq-com.pem
MQTT_AUTH_USERNAME=hfghgfhgf
MQTT_AUTH_PASSWORD=shgfhgf

# Mosquitto connection settings
MQTT_MOSQUITTO_HOST=test.mosquitto.org
MQTT_MOSQUITTO_PORT=1883
MQTT_MOSQUITTO_CLIENT_ID=laravel-mosquitto
MQTT_MOSQUITTO_CLEAN_SESSION=true
MQTT_MOSQUITTO_ENABLE_LOGGING=true
```

## Installation

### Prerequisites

- Go 1.22 or higher
- SSL certificates (if using TLS)

### Building from Source

1. Clone the repository:
   ```
   git clone
   cd MQTTmicroService
   ```

2. Download dependencies:
   ```
   go mod download
   ```

3. Build the application:
   ```
   go build -o mqtt-service main.go
   ```

## Deployment

### Running Locally

1. Set up your environment variables in a `.env` file or export them directly.

2. Run the service:
   ```
   ./mqtt-service
   ```


### Production Deployment

For production deployment, consider the following:

1. Use a process manager like systemd or supervisor to ensure the service stays running.

2. Set up proper logging to a file or a centralized logging system.

3. Use a reverse proxy like Nginx to handle SSL termination and load balancing.

4. Store SSL certificates securely and ensure they are regularly updated.

## API Usage Examples

### Publish a Message

**Endpoint**: `POST /publish`

**Request**:
```json
{
  "topic": "sensors/temperature",
  "payload": {"value": 23.5, "unit": "celsius"},
  "qos": 1,
  "retained": false,
  "broker": "hivemq"
}
```

**Response**:
```json
{
  "status": "success",
  "message": "Message published successfully"
}
```

### Subscribe to a Topic

**Endpoint**: `POST /subscribe`

**Request**:
```json
{
  "topic": "sensors/temperature",
  "qos": 1,
  "broker": "hivemq"
}
```

**Response**:
```json
{
  "status": "success",
  "message": "Subscribed to topic sensors/temperature"
}
```

### Unsubscribe from a Topic

**Endpoint**: `POST /unsubscribe`

**Request**:
```json
{
  "topic": "sensors/temperature",
  "broker": "hivemq"
}
```

**Response**:
```json
{
  "status": "success",
  "message": "Unsubscribed from topic sensors/temperature"
}
```

### Check Status

**Endpoint**: `GET /status`

**Response**:
```json
{
  "status": "ok",
  "brokers": {
    "hivemq": {
      "connected": true,
      "subscriptions": ["sensors/temperature", "sensors/humidity"]
    },
    "mosquitto": {
      "connected": false,
      "subscriptions": []
    }
  },
  "timestamp": "2023-04-27T16:43:42Z"
}
```

### Health Check

**Endpoint**: `GET /healthz`

**Response**:
```json
{
  "status": "ok"
}
```

### Get Metrics

**Endpoint**: `GET /metrics`

**Response**:
```json
{
  "messages": {
    "published": 42,
    "received": 18,
    "failed": 2
  },
  "subscriptions": 5,
  "connections": {
    "attempts": 7,
    "failures": 1,
    "successes": 6,
    "disconnections": 2
  },
  "api": {
    "requests": 156,
    "errors": 3
  },
  "latency": {
    "publish": "15.2ms",
    "subscribe": "22.7ms"
  },
  "last_updated": "2023-04-27T16:43:42Z"
}
```

### View Logs

**Endpoint**: `GET /logs`

**Response**: Plain text log output

### Get Messages from Database

**Endpoint**: `GET /messages?confirmed=false&limit=10`

**Response**:
```json
{
  "status": "success",
  "messages": [
    {
      "id": "1682619845123456789",
      "topic": "sensors/temperature",
      "payload": {"value": 23.5, "unit": "celsius"},
      "qos": 1,
      "retained": false,
      "timestamp": "2023-04-27T16:43:42Z",
      "confirmed": false
    },
    {
      "id": "1682619845987654321",
      "topic": "sensors/humidity",
      "payload": {"value": 45.2, "unit": "percent"},
      "qos": 1,
      "retained": false,
      "timestamp": "2023-04-27T16:43:42Z",
      "confirmed": false
    }
  ],
  "count": 2
}
```

### Get Message by ID

**Endpoint**: `GET /messages/{id}`

**Response**:
```json
{
  "status": "success",
  "message": {
    "id": "1682619845123456789",
    "topic": "sensors/temperature",
    "payload": {"value": 23.5, "unit": "celsius"},
    "qos": 1,
    "retained": false,
    "timestamp": "2023-04-27T16:43:42Z",
    "confirmed": false
  }
}
```

### Confirm Message

**Endpoint**: `POST /messages/{id}/confirm`

**Response**:
```json
{
  "status": "success",
  "message": "Message 1682619845123456789 confirmed"
}
```

### Delete Message

**Endpoint**: `DELETE /messages/{id}`

**Response**:
```json
{
  "status": "success",
  "message": "Message 1682619845123456789 deleted"
}
```

### Delete Confirmed Messages

**Endpoint**: `DELETE /messages/confirmed`

**Response**:
```json
{
  "status": "success",
  "message": "5 confirmed messages deleted",
  "count": 5
}
```
