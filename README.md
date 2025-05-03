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
- Webhook notifications for received messages
- Health check endpoint
- Comprehensive logging

## Architecture

The microservice is built with a clean, modular architecture:

- **Configuration Module**: Loads and validates connection settings from environment variables
- **MQTT Client Manager**: Manages connections to multiple MQTT brokers
- **HTTP API Server**: Exposes endpoints for publishing messages, managing subscriptions, and checking status
- **Webhook Notifier**: Sends HTTP notifications to Laravel when messages are received
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

### Webhook Management Endpoints
- `GET /webhooks`: Get all webhooks
- `POST /webhooks`: Create a new webhook
- `GET /webhooks/{id}`: Get a specific webhook by ID
- `PUT /webhooks/{id}`: Update a webhook
- `DELETE /webhooks/{id}`: Delete a webhook

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

# Webhook settings
WEBHOOK_ENABLED=true
WEBHOOK_URL=https://your-laravel-app.com/api/mqtt/webhook
WEBHOOK_METHOD=POST
WEBHOOK_TIMEOUT=10
WEBHOOK_RETRY_COUNT=3
WEBHOOK_RETRY_DELAY=5
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

### Get All Webhooks

**Endpoint**: `GET /webhooks`

**Response**:
```json
{
  "status": "success",
  "webhooks": [
    {
      "id": "1682619845123456789",
      "name": "Temperature Webhook",
      "url": "https://your-laravel-app.com/api/temperature",
      "method": "POST",
      "topic_filter": "sensors/temperature",
      "enabled": true,
      "headers": {
        "X-API-Key": "your-api-key"
      },
      "timeout": 10,
      "retry_count": 3,
      "retry_delay": 5,
      "created_at": "2023-04-27T16:43:42Z",
      "updated_at": "2023-04-27T16:43:42Z"
    }
  ],
  "count": 1
}
```

### Get Webhook by ID

**Endpoint**: `GET /webhooks/{id}`

**Response**:
```json
{
  "status": "success",
  "webhook": {
    "id": "1682619845123456789",
    "name": "Temperature Webhook",
    "url": "https://your-laravel-app.com/api/temperature",
    "method": "POST",
    "topic_filter": "sensors/temperature",
    "enabled": true,
    "headers": {
      "X-API-Key": "your-api-key"
    },
    "timeout": 10,
    "retry_count": 3,
    "retry_delay": 5,
    "created_at": "2023-04-27T16:43:42Z",
    "updated_at": "2023-04-27T16:43:42Z"
  }
}
```

### Create Webhook

**Endpoint**: `POST /webhooks`

**Request**:
```json
{
  "name": "Temperature Webhook",
  "url": "https://your-laravel-app.com/api/temperature",
  "method": "POST",
  "topic_filter": "sensors/temperature",
  "enabled": true,
  "headers": {
    "X-API-Key": "your-api-key"
  },
  "timeout": 10,
  "retry_count": 3,
  "retry_delay": 5
}
```

**Response**:
```json
{
  "status": "success",
  "message": "Webhook created successfully",
  "webhook": {
    "id": "1682619845123456789",
    "name": "Temperature Webhook",
    "url": "https://your-laravel-app.com/api/temperature",
    "method": "POST",
    "topic_filter": "sensors/temperature",
    "enabled": true,
    "headers": {
      "X-API-Key": "your-api-key"
    },
    "timeout": 10,
    "retry_count": 3,
    "retry_delay": 5,
    "created_at": "2023-04-27T16:43:42Z",
    "updated_at": "2023-04-27T16:43:42Z"
  }
}
```

### Update Webhook

**Endpoint**: `PUT /webhooks/{id}`

**Request**:
```json
{
  "name": "Updated Temperature Webhook",
  "url": "https://your-laravel-app.com/api/temperature/v2",
  "method": "POST",
  "topic_filter": "sensors/+/temperature",
  "enabled": true,
  "headers": {
    "X-API-Key": "your-new-api-key"
  },
  "timeout": 15,
  "retry_count": 5,
  "retry_delay": 10
}
```

**Response**:
```json
{
  "status": "success",
  "message": "Webhook updated successfully",
  "webhook": {
    "id": "1682619845123456789",
    "name": "Updated Temperature Webhook",
    "url": "https://your-laravel-app.com/api/temperature/v2",
    "method": "POST",
    "topic_filter": "sensors/+/temperature",
    "enabled": true,
    "headers": {
      "X-API-Key": "your-new-api-key"
    },
    "timeout": 15,
    "retry_count": 5,
    "retry_delay": 10,
    "created_at": "2023-04-27T16:43:42Z",
    "updated_at": "2023-04-27T16:45:00Z"
  }
}
```

### Delete Webhook

**Endpoint**: `DELETE /webhooks/{id}`

**Response**:
```json
{
  "status": "success",
  "message": "Webhook 1682619845123456789 deleted successfully"
}
```

## Webhook Notifications

The microservice can send webhook notifications to your Laravel application when messages are received on subscribed topics. This allows your Laravel application to react to MQTT messages without having to poll the microservice.

The microservice supports two types of webhook configurations:

1. **Global Webhook**: Configured using environment variables
2. **Database Webhooks**: Created and managed via API endpoints (see [Webhook Management Endpoints](#webhook-management-endpoints))

When a message is received on a subscribed topic, the microservice sends notifications to both the global webhook (if enabled) and any matching webhooks from the database.

### Global Webhook Configuration

The global webhook is configured using environment variables:

```
WEBHOOK_ENABLED=true
WEBHOOK_URL=https://your-laravel-app.com/api/mqtt/webhook
WEBHOOK_METHOD=POST
WEBHOOK_TIMEOUT=10
WEBHOOK_RETRY_COUNT=3
WEBHOOK_RETRY_DELAY=5
```

- `WEBHOOK_ENABLED`: Set to `true` to enable global webhook notifications
- `WEBHOOK_URL`: The URL to send webhook notifications to
- `WEBHOOK_METHOD`: The HTTP method to use (default: `POST`)
- `WEBHOOK_TIMEOUT`: The timeout for webhook requests in seconds (default: `10`)
- `WEBHOOK_RETRY_COUNT`: The number of times to retry failed webhook requests (default: `3`)
- `WEBHOOK_RETRY_DELAY`: The delay between retries in seconds (default: `5`)

### Database Webhooks

In addition to the global webhook, you can create and manage webhooks via API endpoints. These webhooks are stored in the database and can be configured to match specific MQTT topics using wildcards. See the [Webhook Management Endpoints](#webhook-management-endpoints) section for details on how to create and manage database webhooks.

### Webhook Payload

When a message is received on a subscribed topic, the microservice sends a webhook notification to the configured URL with the following payload:

```json
{
  "topic": "sensors/temperature",
  "payload": {"value": 23.5, "unit": "celsius"},
  "qos": 1,
  "timestamp": "2023-04-27T16:43:42Z",
  "broker": "hivemq"
}
```

- `topic`: The MQTT topic the message was received on
- `payload`: The message payload (parsed as JSON if possible, otherwise as a string)
- `qos`: The QoS level of the message
- `timestamp`: The time the message was received
- `broker`: The name of the broker the message was received from

### Laravel Integration

To integrate with Laravel, create a route and controller to handle the webhook notifications:

```php
// routes/api.php
Route::post('/mqtt/webhook', 'MqttWebhookController@handle');
```

```php
// app/Http/Controllers/MqttWebhookController.php
namespace App\Http\Controllers;

use Illuminate\Http\Request;

class MqttWebhookController extends Controller
{
    public function handle(Request $request)
    {
        $topic = $request->input('topic');
        $payload = $request->input('payload');
        $qos = $request->input('qos');
        $timestamp = $request->input('timestamp');
        $broker = $request->input('broker');

        // Process the message
        // For example, dispatch an event
        event(new MqttMessageReceived($topic, $payload, $qos, $timestamp, $broker));

        return response()->json(['status' => 'success']);
    }
}
```
