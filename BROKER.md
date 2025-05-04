# MQTT Broker Documentation

## Overview

The MQTT microservice includes a built-in MQTT broker that can be used alongside the MQTT client functionality. This allows the microservice to act as both a client (connecting to external MQTT brokers) and a server (accepting connections from MQTT clients).

The broker is implemented using the [Mochi MQTT](https://github.com/mochi-mqtt/server) library, a high-performance, feature-rich MQTT server implementation in Go.

## Features

- **Standalone MQTT Broker**: Run your own MQTT broker without needing external services like HiveMQ or Mosquitto
- **TCP and TLS Support**: Secure your MQTT communications with TLS
- **Authentication**: Control access to your broker with username/password authentication
- **Status Monitoring**: Check the status of the broker through the API
- **Configurable**: Easily configure the broker through environment variables

## Configuration

The broker can be configured using the following environment variables:

```
# MQTT Broker settings
MQTT_BROKER_ENABLED=true           # Enable or disable the broker
MQTT_BROKER_HOST=0.0.0.0           # Host to bind to (0.0.0.0 for all interfaces)
MQTT_BROKER_PORT=1883              # Port to listen on (standard MQTT port is 1883)
MQTT_BROKER_TLS_ENABLED=false      # Enable or disable TLS
MQTT_BROKER_TLS_CERT_FILE=certificates/server.crt  # Path to TLS certificate file
MQTT_BROKER_TLS_KEY_FILE=certificates/server.key   # Path to TLS key file
MQTT_BROKER_AUTH_ENABLED=false     # Enable or disable authentication
MQTT_BROKER_ALLOW_ANONYMOUS=true   # Allow anonymous connections when auth is enabled
MQTT_BROKER_CREDENTIALS=user1:password1,user2:password2  # Comma-separated list of username:password pairs
```

## API Endpoints

### Get Broker Status

**Endpoint**: `GET /broker/status`

Returns the status of the built-in MQTT broker.

**Response**:
```json
{
  "status": "ok",
  "broker": {
    "running": true,
    "enabled": true,
    "clients": 0
  }
}
```

## Usage Examples

### Starting the Broker

The broker starts automatically when the microservice starts, if it's enabled in the configuration.

### Connecting to the Broker

You can connect to the broker using any MQTT client. Here are some examples:

#### Using the MQTT CLI

```bash
# Connect to the broker
mqtt sub -h localhost -p 1883 -t "test/topic"

# Publish a message
mqtt pub -h localhost -p 1883 -t "test/topic" -m "Hello, world!"
```

#### Using Mosquitto Clients

```bash
# Subscribe to a topic
mosquitto_sub -h localhost -p 1883 -t "test/topic"

# Publish a message
mosquitto_pub -h localhost -p 1883 -t "test/topic" -m "Hello, world!"
```

#### Using the Microservice's API

You can also use the microservice's API to publish messages to the broker:

```bash
# Publish a message
curl -X POST http://localhost:8080/publish \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "test/topic",
    "payload": "Hello, world!",
    "qos": 0,
    "retained": false,
    "broker": "internal"  # Use "internal" to publish to the built-in broker
  }'
```

### Checking Broker Status

You can check the status of the broker using the API:

```bash
curl -X GET http://localhost:8080/broker/status
```

## Security Considerations

When deploying the broker in production, consider the following security best practices:

1. **Enable TLS**: Always use TLS in production to encrypt MQTT traffic.
2. **Enable Authentication**: Require clients to authenticate with username and password.
3. **Use Strong Credentials**: Use strong, unique passwords for each client.
4. **Restrict Access**: Use firewall rules to restrict access to the broker port.
5. **Monitor Connections**: Regularly check the broker status to monitor connections.

## Limitations

The current implementation has the following limitations:

1. **No Persistent Sessions**: Client sessions are not persisted across broker restarts.
2. **No Retained Messages**: Retained messages are not supported.
3. **No Will Messages**: Will messages are not supported.
4. **No QoS 2**: Only QoS 0 and 1 are fully supported.
5. **No Access Control Lists (ACLs)**: Fine-grained access control is not implemented.

## Troubleshooting

### Common Issues

1. **Broker Won't Start**:
   - Check if the port is already in use by another application.
   - Verify that the TLS certificate and key files exist and are valid.

2. **Clients Can't Connect**:
   - Ensure the broker is running (check `/broker/status` endpoint).
   - Verify that the client is using the correct host and port.
   - If authentication is enabled, ensure the client is providing valid credentials.

3. **TLS Connection Issues**:
   - Verify that the TLS certificate and key files are valid.
   - Ensure the client is configured to use TLS.
   - Check that the client trusts the broker's certificate.

### Logs

Check the microservice logs for more information about broker-related issues:

```bash
curl -X GET http://localhost:8080/logs
```

## Advanced Configuration

For advanced configuration options not exposed through environment variables, you may need to modify the broker code directly. The broker is implemented in the `internal/broker` package.