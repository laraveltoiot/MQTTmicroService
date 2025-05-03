# MQTT Microservice Architecture

This document provides a visual representation of the MQTT Microservice architecture using Mermaid diagrams. It illustrates the components, their relationships, and the communication flow between the microservice and Laravel.

## Overall Architecture

The MQTT Microservice is built with a clean, modular architecture consisting of several key components that work together to provide MQTT functionality to Laravel applications.

```mermaid
graph TD
    subgraph "MQTT Microservice"
        A[Config] --> B[Logger]
        A --> C[MQTT Manager]
        A --> D[API Server]
        A --> E[Database]
        A --> F[Auth Service]
        A --> G[Metrics Collector]
        
        B --> C
        B --> D
        B --> E
        B --> F
        B --> G
        
        C --> D
        E --> D
        F --> D
        G --> D
        
        C --> H[MQTT Client 1]
        C --> I[MQTT Client 2]
        C --> J[MQTT Client n]
        
        H --> K[MQTT Broker 1]
        I --> L[MQTT Broker 2]
        J --> M[MQTT Broker n]
    end
    
    subgraph "Laravel Application"
        N[Laravel Backend]
    end
    
    N <--> D
    K <--> H
    L <--> I
    M <--> J
```

### Component Description

1. **Config**: Loads and validates configuration from environment variables
2. **Logger**: Provides structured logging throughout the application
3. **MQTT Manager**: Manages connections to multiple MQTT brokers
4. **API Server**: Exposes HTTP endpoints for Laravel to interact with
5. **Database**: Stores messages and webhook configurations
6. **Auth Service**: Handles API key authentication
7. **Metrics Collector**: Tracks performance and usage metrics
8. **MQTT Clients**: Connect to MQTT brokers and handle messaging
9. **MQTT Brokers**: External MQTT servers (HiveMQ, Mosquitto, etc.)

## Communication with Laravel

The MQTT Microservice communicates with Laravel in two ways:
1. Laravel makes HTTP requests to the microservice's API endpoints
2. The microservice sends webhook notifications to Laravel when messages are received

```mermaid
sequenceDiagram
    participant Laravel as Laravel Application
    participant API as MQTT Microservice API
    participant MQTT as MQTT Manager
    participant Broker as MQTT Broker
    
    %% Laravel publishing a message
    Laravel->>API: POST /publish (topic, payload, QoS)
    API->>MQTT: Publish message
    MQTT->>Broker: Publish to topic
    API->>Laravel: Success response
    
    %% Laravel subscribing to a topic
    Laravel->>API: POST /subscribe (topic, QoS)
    API->>MQTT: Subscribe to topic
    MQTT->>Broker: Subscribe to topic
    API->>Laravel: Success response
    
    %% Message received from broker
    Broker->>MQTT: Message on subscribed topic
    MQTT->>API: Process received message
    API->>Laravel: Webhook notification
    Laravel->>API: Webhook acknowledgment
```

## Data Flow for Key Operations

### Publishing a Message

```mermaid
flowchart TD
    A[Laravel] -->|1. POST /publish| B[API Server]
    B -->|2. Validate request| C{Valid?}
    C -->|No| D[Return error]
    C -->|Yes| E[Get MQTT client]
    E -->|3. Publish message| F[MQTT Broker]
    E -->|4. Store in database| G[(Database)]
    E -->|5. Return success| A
```

### Subscribing to a Topic

```mermaid
flowchart TD
    A[Laravel] -->|1. POST /subscribe| B[API Server]
    B -->|2. Validate request| C{Valid?}
    C -->|No| D[Return error]
    C -->|Yes| E[Get MQTT client]
    E -->|3. Subscribe to topic| F[MQTT Broker]
    E -->|4. Register message handler| G[Message Handler]
    E -->|5. Return success| A
    
    F -->|6. Message on topic| G
    G -->|7. Process message| H{Webhook configured?}
    H -->|Yes| I[Send webhook notification]
    I -->|8. HTTP POST| A
    H -->|No| J[Log message]
```

### Webhook Notifications

```mermaid
flowchart TD
    A[MQTT Broker] -->|1. Message on topic| B[MQTT Client]
    B -->|2. Process message| C[Message Handler]
    C -->|3. Find matching webhooks| D[(Database)]
    D -->|4. Return webhooks| C
    C -->|5. For each webhook| E[Send notification]
    E -->|6. HTTP POST| F[Laravel]
    F -->|7. Process message| G{Success?}
    G -->|Yes| H[Return 200 OK]
    G -->|No| I[Return error]
    I -->|8. Retry if configured| E
```

## Database Schema

The microservice uses a database to store messages and webhook configurations. The schema is as follows:

```mermaid
erDiagram
    MESSAGES {
        string id PK
        string topic
        blob payload
        int qos
        bool retained
        datetime timestamp
        bool confirmed
    }
    
    WEBHOOKS {
        string id PK
        string name
        string url
        string method
        string topic_filter
        bool enabled
        json headers
        int timeout
        int retry_count
        int retry_delay
        datetime created_at
        datetime updated_at
    }
```

## Deployment Architecture

The MQTT Microservice can be deployed in various ways, depending on the requirements. Here's a typical deployment architecture:

```mermaid
graph TD
    subgraph "Cloud/Server"
        A[Laravel Application] <--> B[MQTT Microservice]
        B <--> C[(Database)]
        B <--> D[MQTT Broker]
    end
    
    subgraph "IoT Devices"
        E[Device 1] <--> D
        F[Device 2] <--> D
        G[Device n] <--> D
    end
```

This architecture allows Laravel to offload all MQTT communication to the microservice, which handles the complexities of MQTT protocols, connection management, and message processing.