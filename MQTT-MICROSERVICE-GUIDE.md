# MQTT Microservice User Guide

## Introduction

The MQTT Microservice is a professional Go application that serves as a client for Laravel IoT Cloud backends. It handles all MQTT communication (subscribe, publish, reconnects, handling SSL certificates), offloading the MQTT layer from the Laravel app.

This guide explains how to use the microservice, including its API endpoints, logging system, telemetry, and configuration options.

## Table of Contents

- [Architecture](#architecture)
- [API Endpoints](#api-endpoints)
  - [Publish Messages](#publish-messages)
  - [Subscribe to Topics](#subscribe-to-topics)
  - [Unsubscribe from Topics](#unsubscribe-from-topics)
  - [Check Status](#check-status)
  - [Health Check](#health-check)
  - [Database Operations](#database-operations)
- [Webhook Notifications](#webhook-notifications)
  - [Configuration](#webhook-configuration)
  - [Payload Format](#webhook-payload-format)
  - [Laravel Integration](#laravel-integration)
- [Logging System](#logging-system)
- [Telemetry and Metrics](#telemetry-and-metrics)
- [Configuration](#configuration)
  - [Environment Variables](#environment-variables)
  - [SSL/TLS Configuration](#ssltls-configuration)
  - [Database Configuration](#database-configuration)
  - [Webhook Configuration](#webhook-configuration-1)
- [Authentication](#authentication)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)

## Architecture

The microservice is built with a clean, modular architecture:

- **Configuration Module**: Loads and validates connection settings from environment variables
- **MQTT Client Manager**: Manages connections to multiple MQTT brokers
- **HTTP API Server**: Exposes endpoints for publishing messages, managing subscriptions, and checking status
- **Webhook Notifier**: Sends HTTP notifications to Laravel when messages are received on subscribed topics
- **Logging Utility**: Provides consistent logging throughout the application
- **Metrics Collector**: Tracks performance and usage metrics
- **Authentication Module**: Secures API endpoints with API key authentication

## API Endpoints

The microservice exposes the following HTTP API endpoints:

### Publish Messages

**Endpoint**: `POST /publish`

Publishes a message to a specified MQTT topic.

**Request Body**:
```json
{
  "topic": "sensors/temperature",
  "payload": {"value": 23.5, "unit": "celsius"},
  "qos": 1,
  "retained": false,
  "broker": "hivemq"
}
```

The `payload` field can be:
- A string
- A number
- A boolean
- A JSON object or array

**Response (Success)**:
```json
{
  "status": "success",
  "message": "Message published successfully"
}
```

**Response (Error)**:
```json
{
  "status": "error",
  "message": "Failed to publish message: [error details]"
}
```

**Example (using curl)**:
```bash
curl -X POST http://localhost:8080/publish \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "sensors/temperature",
    "payload": {"value": 23.5, "unit": "celsius"},
    "qos": 1,
    "retained": false
  }'
```

### Subscribe to Topics

**Endpoint**: `POST /subscribe`

Subscribes to a specified MQTT topic.

**Request Body**:
```json
{
  "topic": "sensors/temperature",
  "qos": 1,
  "broker": "hivemq"
}
```

**Response (Success)**:
```json
{
  "status": "success",
  "message": "Subscribed to topic sensors/temperature"
}
```

**Response (Error)**:
```json
{
  "status": "error",
  "message": "Failed to subscribe to topic: [error details]"
}
```

**Example (using curl)**:
```bash
curl -X POST http://localhost:8080/subscribe \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "sensors/temperature",
    "qos": 1
  }'
```

### Unsubscribe from Topics

**Endpoint**: `POST /unsubscribe`

Unsubscribes from a specified MQTT topic.

**Request Body**:
```json
{
  "topic": "sensors/temperature",
  "broker": "hivemq"
}
```

**Response (Success)**:
```json
{
  "status": "success",
  "message": "Unsubscribed from topic sensors/temperature"
}
```

**Response (Error)**:
```json
{
  "status": "error",
  "message": "Failed to unsubscribe from topic: [error details]"
}
```

**Example (using curl)**:
```bash
curl -X POST http://localhost:8080/unsubscribe \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "sensors/temperature"
  }'
```

### Check Status

**Endpoint**: `GET /status`

Returns the status of all MQTT connections.

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

The `status` field can be:
- `ok`: All brokers are connected
- `partial`: Some brokers are connected
- `no_clients`: No broker clients are available

**Example (using curl)**:
```bash
curl -X GET http://localhost:8080/status
```

### Health Check

**Endpoint**: `GET /healthz`

Simple health check endpoint that returns a 200 OK response if the service is running.

**Response**:
```json
{
  "status": "ok"
}
```

**Example (using curl)**:
```bash
curl -X GET http://localhost:8080/healthz
```

### Webhook Management

The microservice provides endpoints for managing webhooks. Webhooks allow you to configure HTTP callbacks that are triggered when messages are received on specific MQTT topics.

#### Get All Webhooks

**Endpoint**: `GET /webhooks`

Retrieves all configured webhooks.

**Query Parameters**:
- `limit` (optional): Maximum number of webhooks to return, default is 100

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

**Example (using curl)**:
```bash
curl -X GET "http://localhost:8080/webhooks?limit=10"
```

#### Get Webhook by ID

**Endpoint**: `GET /webhooks/{id}`

Retrieves a specific webhook by its ID.

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

**Example (using curl)**:
```bash
curl -X GET http://localhost:8080/webhooks/1682619845123456789
```

#### Create Webhook

**Endpoint**: `POST /webhooks`

Creates a new webhook.

**Request Body**:
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

**Example (using curl)**:
```bash
curl -X POST http://localhost:8080/webhooks \
  -H "Content-Type: application/json" \
  -d '{
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
  }'
```

#### Update Webhook

**Endpoint**: `PUT /webhooks/{id}`

Updates an existing webhook.

**Request Body**:
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

**Example (using curl)**:
```bash
curl -X PUT http://localhost:8080/webhooks/1682619845123456789 \
  -H "Content-Type: application/json" \
  -d '{
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
  }'
```

#### Delete Webhook

**Endpoint**: `DELETE /webhooks/{id}`

Deletes a webhook.

**Response**:
```json
{
  "status": "success",
  "message": "Webhook 1682619845123456789 deleted successfully"
}
```

**Example (using curl)**:
```bash
curl -X DELETE http://localhost:8080/webhooks/1682619845123456789
```

### Database Operations

The microservice includes a database integration that stores MQTT messages and allows Laravel to confirm receipt of messages. This ensures that messages are not lost if Laravel is temporarily unavailable.

#### Get Messages

**Endpoint**: `GET /messages`

Retrieves messages from the database.

**Query Parameters**:
- `confirmed` (optional): Set to "true" to get confirmed messages, default is "false" (unconfirmed messages)
- `limit` (optional): Maximum number of messages to return, default is 100

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

**Example (using curl)**:
```bash
curl -X GET "http://localhost:8080/messages?confirmed=false&limit=10"
```

#### Get Message by ID

**Endpoint**: `GET /messages/{id}`

Retrieves a specific message by its ID.

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

**Example (using curl)**:
```bash
curl -X GET http://localhost:8080/messages/1682619845123456789
```

#### Confirm Message

**Endpoint**: `POST /messages/{id}/confirm`

Marks a message as confirmed, indicating that Laravel has successfully processed it.

**Response**:
```json
{
  "status": "success",
  "message": "Message 1682619845123456789 confirmed"
}
```

**Example (using curl)**:
```bash
curl -X POST http://localhost:8080/messages/1682619845123456789/confirm
```

#### Delete Message

**Endpoint**: `DELETE /messages/{id}`

Deletes a specific message from the database.

**Response**:
```json
{
  "status": "success",
  "message": "Message 1682619845123456789 deleted"
}
```

**Example (using curl)**:
```bash
curl -X DELETE http://localhost:8080/messages/1682619845123456789
```

#### Delete Confirmed Messages

**Endpoint**: `DELETE /messages/confirmed`

Deletes all confirmed messages from the database.

**Response**:
```json
{
  "status": "success",
  "message": "5 confirmed messages deleted",
  "count": 5
}
```

**Example (using curl)**:
```bash
curl -X DELETE http://localhost:8080/messages/confirmed
```

## Webhook Notifications

The microservice can send webhook notifications to your Laravel application when messages are received on subscribed topics. This allows your Laravel application to react to MQTT messages without having to poll the microservice.

The microservice supports two types of webhook configurations:

1. **Global Webhook**: Configured using environment variables
2. **Database Webhooks**: Created and managed via API endpoints

When a message is received on a subscribed topic, the microservice sends notifications 
to both the global webhook (if enabled) and any matching webhooks from the database.

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

In addition to the global webhook, you can create and manage webhooks via API endpoints. These webhooks are stored in the database and can be configured to match specific MQTT topics using wildcards.

Database webhooks are managed using the following API endpoints:

- `GET /webhooks`: List all webhooks
- `GET /webhooks/{id}`: Get a specific webhook
- `POST /webhooks`: Create a new webhook
- `PUT /webhooks/{id}`: Update a webhook
- `DELETE /webhooks/{id}`: Delete a webhook

When creating a webhook, you can specify:

- `name`: A descriptive name for the webhook
- `url`: The URL to send webhook notifications to
- `method`: The HTTP method to use (default: `POST`)
- `topic_filter`: The MQTT topic filter to match (supports wildcards like `+` and `#`)
- `enabled`: Whether the webhook is active (default: `true`)
- `headers`: Custom HTTP headers to include in the webhook request
- `timeout`: The timeout for webhook requests in seconds (default: `10`)
- `retry_count`: The number of times to retry failed webhook requests (default: `3`)
- `retry_delay`: The delay between retries in seconds (default: `5`)

Example of creating a webhook:

```bash
curl -X POST http://localhost:8080/webhooks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Temperature Webhook",
    "url": "https://your-laravel-app.com/api/temperature",
    "method": "POST",
    "topic_filter": "sensors/+/temperature",
    "enabled": true,
    "headers": {
      "X-API-Key": "your-api-key"
    },
    "timeout": 10,
    "retry_count": 3,
    "retry_delay": 5
  }'
```

### Webhook Payload Format

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

### Metrics

**Endpoint**: `GET /metrics`

Returns detailed metrics about the MQTT microservice, including message counts, connection statistics, and performance metrics.

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

**Example (using curl)**:
```bash
curl -X GET http://localhost:8080/metrics
```

### Logs

**Endpoint**: `GET /logs`

Returns the contents of the log file. By default, it returns the main log file, but you can specify a different log file using the `file` query parameter.

**Query Parameters**:
- `file` (optional): The name of the log file to view (default: `mqtt-service.log`)
- `lines` (optional): The number of lines to return (not implemented yet)

**Response**:
```
time="2023-04-27 16:33:15" level=info msg="Starting MQTT microservice"
time="2023-04-27 16:33:16" level=info msg="Connected to default MQTT broker" broker=hivemq
time="2023-04-27 16:33:16" level=info msg="HTTP server started" addr=:8080
time="2023-04-27 16:43:42" level=info msg="Received message" topic="sensors/temperature" payload="{\"value\":23.5,\"unit\":\"celsius\"}" qos=1
```

**Example (using curl)**:
```bash
curl -X GET http://localhost:8080/logs
```

**Example (specifying a log file)**:
```bash
curl -X GET "http://localhost:8080/logs?file=error.log"
```

## Logging System

The microservice uses a structured logging system based on logrus. Logs can be configured to output in text or JSON format and can be directed to the console, a file, or both.

### Log Levels

The following log levels are supported (from lowest to highest severity):
- `debug`: Detailed debugging information
- `info`: Informational messages
- `warn`: Warning messages
- `error`: Error messages
- `fatal`: Fatal errors that cause the application to exit

### Log Format

Logs can be formatted as:
- `text`: Human-readable format (default)
- `json`: JSON format for machine processing

### Log Configuration

Logging is configured through command-line flags or environment variables:

**Command-line Flags**:
- `--log-level`: Sets the minimum log level (default: `info`)
- `--log-format`: Sets the log format (`text` or `json`, default: `text`)
- `--log-file`: Sets the log file path (default: `mqtt-service.log`)
- `--file-logging`: Enables or disables logging to a file (default: `true`)

**Environment Variables**:
- `LOG_LEVEL`: Sets the minimum log level (default: `info`)
- `LOG_FORMAT`: Sets the log format (`text` or `json`, default: `text`)

### File Logging

By default, the microservice logs to both the console and a file named `mqtt-service.log` in the current directory. You can:

1. Disable file logging with the `--file-logging=false` flag
2. Change the log file path with the `--log-file` flag
3. View the logs through the `/logs` API endpoint
4. Access the log file directly on the filesystem

If file logging fails (e.g., due to permission issues), the microservice will fall back to console-only logging.

### Example Log Output (Text Format)

```
time="2023-04-27 16:33:15" level=info msg="Starting MQTT microservice"
time="2023-04-27 16:33:16" level=info msg="Connected to default MQTT broker" broker=hivemq
time="2023-04-27 16:33:16" level=info msg="HTTP server started" addr=:8080
time="2023-04-27 16:43:42" level=info msg="Received message" topic="sensors/temperature" payload="{\"value\":23.5,\"unit\":\"celsius\"}" qos=1
```

### Example Log Output (JSON Format)

When using JSON format, each log entry is a separate JSON object on its own line:

```
{"broker":"hivemq","level":"info","msg":"Connected to default MQTT broker","time":"2023-04-27 16:33:16"}
{"addr":":8080","level":"info","msg":"HTTP server started","time":"2023-04-27 16:33:16"}
{"level":"info","msg":"Received message","payload":"{\"value\":23.5,\"unit\":\"celsius\"}","qos":1,"time":"2023-04-27 16:43:42","topic":"sensors/temperature"}
```

## Telemetry and Metrics

The microservice includes a metrics collection system that tracks various performance and usage metrics.

### Available Metrics

The following metrics are collected:

**Message Metrics**:
- Published messages count
- Received messages count
- Failed publishes count
- Subscription count

**Connection Metrics**:
- Connection attempts
- Connection failures
- Connection successes
- Disconnections

**API Metrics**:
- API requests count
- API errors count

**Performance Metrics**:
- Publish latency (average)
- Subscribe latency (average)

### Accessing Metrics

Metrics are available through the `/metrics` API endpoint, which returns a JSON object containing all collected metrics. This endpoint can be used for monitoring the microservice's performance and usage.

**Example**:
```bash
curl -X GET http://localhost:8080/metrics
```

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

The metrics are updated in real-time as events occur in the microservice.

## Configuration

### Environment Variables

The microservice is configured using environment variables. These can be set directly in the environment or through a `.env` file.

**Core Settings**:
- `MQTT_DEFAULT_CONNECTION`: The default broker to use (required)
- `HTTP_SERVER_PORT`: The port for the HTTP server (default: `8080`)
- `LOG_LEVEL`: The minimum log level (default: `info`)
- `LOG_FORMAT`: The log format (default: `text`)

**Broker Settings**:
For each broker (e.g., `hivemq`, `mosquitto`), the following variables are used:
- `MQTT_[BROKER]_HOST`: The broker hostname
- `MQTT_[BROKER]_PORT`: The broker port
- `MQTT_[BROKER]_CLIENT_ID`: The client ID to use
- `MQTT_[BROKER]_CLEAN_SESSION`: Whether to use a clean session (`true` or `false`)
- `MQTT_[BROKER]_ENABLE_LOGGING`: Whether to enable logging for this broker (`true` or `false`)
- `MQTT_[BROKER]_LOG_CHANNEL`: The log channel to use

**TLS Settings** (applied to all brokers):
- `MQTT_TLS_ENABLED`: Whether to enable TLS (`true` or `false`)
- `MQTT_TLS_VERIFY_PEER`: Whether to verify the peer certificate (`true` or `false`)
- `MQTT_TLS_CA_FILE`: The path to the CA certificate file

**Authentication Settings** (applied to all brokers):
- `MQTT_AUTH_USERNAME`: The username for broker authentication
- `MQTT_AUTH_PASSWORD`: The password for broker authentication

**API Authentication Settings**:
- `API_KEY_ENABLED`: Whether to enable API key authentication (`true` or `false`)
- `API_KEYS`: Comma-separated list of valid API keys

### SSL/TLS Configuration

To use SSL/TLS with the MQTT brokers:

1. Set `MQTT_TLS_ENABLED=true`
2. Optionally set `MQTT_TLS_VERIFY_PEER=true` to verify the broker's certificate
3. If using certificate verification, set `MQTT_TLS_CA_FILE` to the path of the CA certificate file

Example:
```
MQTT_TLS_ENABLED=true
MQTT_TLS_VERIFY_PEER=true
MQTT_TLS_CA_FILE=certificates/www-hivemq-com.pem
```

### Database Configuration

The microservice supports two database backends for storing MQTT messages:

1. **SQLite** (default): A lightweight, file-based database suitable for small to medium deployments
2. **MongoDB**: A NoSQL database suitable for larger deployments with high message volumes

#### Enabling Database Storage

To enable database storage, set the `DB_CONNECTION` environment variable to either `sqlite` or `mongodb`:

```
# Use SQLite (default)
DB_CONNECTION=sqlite
DB_PATH=mqtt-messages.db

# Or use MongoDB
# DB_CONNECTION=mongodb
# DB_PORT=27020
# DB_URI=mongodb://127.0.0.1:27020/laravel
# DB_DATABASE=mqtt_messages
# DB_USERNAME=
# DB_PASSWORD=
```

#### SQLite Configuration

For SQLite, you only need to specify the database file path:

```
DB_CONNECTION=sqlite
DB_PATH=mqtt-messages.db
```

If `DB_PATH` is not specified, the default path `mqtt-messages.db` in the current directory will be used.

#### MongoDB Configuration

For MongoDB, you can either specify a URI or individual connection parameters:

```
DB_CONNECTION=mongodb
DB_URI=mongodb://127.0.0.1:27020/laravel
```

Or:

```
DB_CONNECTION=mongodb
DB_PORT=27020
DB_DATABASE=mqtt_messages
DB_USERNAME=username
DB_PASSWORD=password
```

If both `DB_URI` and individual parameters are specified, the URI takes precedence.

#### How Database Storage Works

1. When a message is published via the MQTT client, it is automatically stored in the database with `confirmed=false`
2. Laravel can retrieve unconfirmed messages via the `/messages` endpoint
3. After processing a message, Laravel should confirm receipt via the `/messages/{id}/confirm` endpoint
4. Confirmed messages can be deleted manually via the `/messages/confirmed` endpoint or will be automatically cleaned up based on retention policies (if configured)

This ensures that messages are not lost if Laravel is temporarily unavailable, as they will remain in the database until explicitly confirmed.

### Webhook Configuration

The microservice can send webhook notifications to your Laravel application when messages are received on subscribed topics. This allows your Laravel application to react to MQTT messages without having to poll the microservice.

To configure webhook notifications, set the following environment variables:

```
# Webhook settings
WEBHOOK_ENABLED=true
WEBHOOK_URL=https://your-laravel-app.com/api/mqtt/webhook
WEBHOOK_METHOD=POST
WEBHOOK_TIMEOUT=10
WEBHOOK_RETRY_COUNT=3
WEBHOOK_RETRY_DELAY=5
```

- `WEBHOOK_ENABLED`: Set to `true` to enable webhook notifications
- `WEBHOOK_URL`: The URL to send webhook notifications to
- `WEBHOOK_METHOD`: The HTTP method to use (default: `POST`)
- `WEBHOOK_TIMEOUT`: The timeout for webhook requests in seconds (default: `10`)
- `WEBHOOK_RETRY_COUNT`: The number of times to retry failed webhook requests (default: `3`)
- `WEBHOOK_RETRY_DELAY`: The delay between retries in seconds (default: `5`)

When a message is received on a subscribed topic, the microservice will send a webhook notification to the configured URL with a JSON payload containing the message details. See the [Webhook Notifications](#webhook-notifications) section for more information.

## Authentication

The microservice supports API key authentication to secure the API endpoints.

### Enabling API Key Authentication

1. Set `API_KEY_ENABLED=true` in your environment or `.env` file
2. Set `API_KEYS` to a comma-separated list of valid API keys

Example:
```
API_KEY_ENABLED=true
API_KEYS=key1,key2,key3
```

When `API_KEY_ENABLED` is set to `false`, API key authentication is disabled, and clients can make requests without providing an API key. This is useful for development or testing environments where security is not a concern.

Example:
```
API_KEY_ENABLED=false
API_KEYS=key1,key2,key3
```

### Using API Keys

When API key authentication is enabled, clients must include a valid API key in their requests. This can be done in three ways:

1. Using the `X-API-Key` header:
```
X-API-Key: key1
```

2. Using the `api_key` query parameter:
```
http://localhost:8080/status?api_key=key1
```

3. Using the `Authorization` header with Bearer token:
```
Authorization: Bearer key1
```

## Testing

### Testing the API

You can test the API endpoints using tools like curl, Postman, or Insomnia.

**Example: Testing the Health Check Endpoint**
```bash
curl -X GET http://localhost:8080/healthz
```

**Example: Testing the Publish Endpoint**
```bash
curl -X POST http://localhost:8080/publish \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "test/topic",
    "payload": "Hello, world!",
    "qos": 0,
    "retained": false
  }'
```

**Example: Testing with Bearer Token Authentication**
```bash
curl -X POST http://localhost:8080/publish \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer 1212122" \
  -d '{
    "topic": "test/topic",
    "payload": "Hello, world!",
    "qos": 0,
    "retained": false
  }'
```

### Testing MQTT Communication

To test MQTT communication, you can use MQTT client tools like:
- [MQTT Explorer](http://mqtt-explorer.com/)
- [MQTT.fx](https://mqttfx.jensd.de/)
- [Mosquitto clients](https://mosquitto.org/download/)

**Example: Using Mosquitto Client to Subscribe to a Topic**
```bash
mosquitto_sub -h test.mosquitto.org -t "test/topic" -v
```

**Example: Using Mosquitto Client to Publish to a Topic**
```bash
mosquitto_pub -h test.mosquitto.org -t "test/topic" -m "Hello from Mosquitto!"
```

### Testing Database Operations

You can test the database operations using tools like curl, Postman, or Insomnia.

**Example: Publishing a Message and Checking Database Storage**

1. Publish a message:
```bash
curl -X POST http://localhost:8080/publish \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "test/database",
    "payload": {"value": 42, "unit": "answer"},
    "qos": 1,
    "retained": false
  }'
```

2. Retrieve unconfirmed messages:
```bash
curl -X GET "http://localhost:8080/messages?confirmed=false"
```

3. Confirm a message:
```bash
curl -X POST http://localhost:8080/messages/1682619845123456789/confirm
```

4. Delete confirmed messages:
```bash
curl -X DELETE http://localhost:8080/messages/confirmed
```

**Example: Testing with Bearer Token Authentication**

If API key authentication is enabled, include the Authorization header:

```bash
curl -X GET "http://localhost:8080/messages" \
  -H "Authorization: Bearer 1212122"
```

## Troubleshooting

### Common Issues

**Issue: Cannot connect to MQTT broker**
- Check that the broker hostname and port are correct
- Verify that the username and password are correct
- If using TLS, ensure the CA certificate is valid and accessible
- Check network connectivity to the broker

**Issue: API returns "Unauthorized" error**
- If API key authentication is enabled, ensure you're providing a valid API key
- Check that the API key is correctly formatted and included in the request
- Try different authentication methods: X-API-Key header, api_key query parameter, or Authorization header with Bearer token

**Issue: Messages not being published**
- Verify that the client is connected to the broker (check `/status` endpoint)
- Ensure the topic is correctly formatted
- Check that the QoS level is appropriate for your use case

**Issue: Not receiving subscribed messages**
- Verify that the subscription was successful
- Check that the topic pattern matches the published messages
- Ensure the broker is configured to allow subscriptions

**Issue: Database connection fails**
- Check that the database configuration is correct in the .env file
- For SQLite, ensure the directory for the database file exists and is writable
- For MongoDB, verify that the MongoDB server is running and accessible
- Check the logs for specific error messages related to database connection

**Issue: Messages not appearing in database**
- Verify that database storage is enabled (`DB_CONNECTION` is set)
- Check that messages are being published successfully
- Examine the logs for any errors during message storage
- Verify that the database is accessible and has sufficient space

**Issue: Cannot confirm or delete messages**
- Ensure you're using the correct message ID
- Check that the database is accessible
- Verify that the message exists in the database
- Check the logs for specific error messages

**Issue: Webhook notifications not being sent**
- Verify that webhook notifications are enabled (`WEBHOOK_ENABLED=true`)
- Check that the webhook URL is correctly configured (`WEBHOOK_URL`)
- Ensure the webhook URL is accessible from the microservice
- Check the logs for webhook-related errors
- Verify that your Laravel application is correctly handling the webhook requests

**Issue: Webhook notifications failing with timeout errors**
- Check that the webhook URL is responding within the configured timeout (`WEBHOOK_TIMEOUT`)
- Increase the timeout value if necessary
- Ensure your Laravel application is processing the webhook requests efficiently
- Consider increasing the retry count (`WEBHOOK_RETRY_COUNT`) and delay (`WEBHOOK_RETRY_DELAY`) for unreliable connections

### Checking Logs

The microservice logs can provide valuable information for troubleshooting:

1. Use the `/logs` API endpoint to view the logs:
   ```bash
   curl -X GET http://localhost:8080/logs
   ```

2. Check the console output if running in the foreground

3. Access the log file directly (default: `mqtt-service.log` in the current directory)

4. Increase the log level to `debug` for more detailed information:
   ```bash
   # Using command-line flag
   ./mqtt-service --log-level=debug

   # Or using environment variable
   LOG_LEVEL=debug ./mqtt-service
   ```

5. If you need to check a specific log file, use the `file` query parameter:
   ```bash
   curl -X GET "http://localhost:8080/logs?file=error.log"
   ```
