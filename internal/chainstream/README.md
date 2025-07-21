# ChainStream Package

The `chainstream` package provides a robust, multi-blockchain block streaming service with built-in resilience features. It serves as the core monitoring component within the blockwatch project, designed to observe multiple blockchain networks simultaneously and handle failures gracefully.

## Package Overview

ChainStream is a streaming service that subscribes to blockchain networks and emits observed blocks through a unified interface. It abstracts away the complexities of network failures, retry logic, and checkpoint management, providing a clean stream of blockchain events to consumers.

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                    ChainStream Service                      │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │   Blockchain    │  │   Checkpoint    │  │    Retry     │ │
│  │   Interface     │  │   Storage       │  │   Handler    │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                    Event Processing                         │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │  Subscription   │  │   Dispatch      │  │   Failure    │ │
│  │   Manager       │  │   Notifier      │  │   Handler    │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                     Output Stream                           │
│              ┌─────────────────────────────┐                │
│              │      ObservedBlock          │                │
│              │       Channel               │                │
│              └─────────────────────────────┘                │
└─────────────────────────────────────────────────────────────┘
```

## Key Interfaces

### Service Interface
```go
type Service interface {
    // Start begins the block observation process and returns a channel of observed blocks
    // Returns ErrServiceAlreadyStarted if called more than once
    Start(ctx context.Context) (<-chan ObservedBlock, error)
    
    // Close terminates all background processes and cleans up resources
    Close()
}
```

### Blockchain Interface
The package depends on implementations of the `Blockchain` interface to provide blockchain data:

```go
type Blockchain interface {
    // FetchBlockByHeight retrieves a specific block by its height
    FetchBlockByHeight(ctx context.Context, height types.Hex) (Block, error)
    
    // Subscribe streams blocks starting from the specified height
    Subscribe(ctx context.Context, fromHeight types.Hex) (<-chan BlockchainEvent, error)
}
```

### CheckpointStorage Interface
Optional interface for persisting processing progress:

```go
type CheckpointStorage interface {
    // SaveCheckpoint records the latest processed block height for a network
    SaveCheckpoint(ctx context.Context, network string, height types.Hex) error
    
    // LoadLatestCheckpoint retrieves the last saved checkpoint for a network
    // Returns ErrNoCheckpointFound if no checkpoint exists for the network
    LoadLatestCheckpoint(ctx context.Context, network string) (types.Hex, error)
}
```

### DispatchFailureNotifier Interface
Interface for handling unrecoverable dispatch failures:

```go
type DispatchFailureNotifier interface {
    // NotifyDispatchFailure is called when a dispatch failure occurs that cannot be recovered
    // It receives contextual information and the failure details
    // If an error is returned, it will be logged by the service
    NotifyDispatchFailure(ctx context.Context, failure BlockDispatchFailure) error
}
```

## Data Types

### ObservedBlock
The primary output type containing a blockchain block with network context:

```go
type ObservedBlock struct {
    Network string // Network identifier (e.g., "ethereum", "polygon")
    Block          // Embedded block data
}
```

### Block
Represents a blockchain block:

```go
type Block struct {
    Height       types.Hex     // Block height as hex string
    Hash         string        // Unique block hash
    Transactions []Transaction // List of transactions in the block
}
```

### Transaction
Represents a blockchain transaction:

```go
type Transaction struct {
    Hash string // Unique transaction hash identifier
    From string // Sender address
    To   string // Recipient address
}
```

### BlockchainEvent
Events emitted by blockchain implementations:

```go
type BlockchainEvent struct {
    Height types.Hex // Block height (always present)
    Block  Block     // Block data (empty if Err is set)
    Err    error     // Error if block retrieval failed
}
```

### BlockDispatchFailure
Represents failures in block processing:

```go
type BlockDispatchFailure struct {
    Network string    // Network where failure occurred
    Height  types.Hex // Block height that failed
    Errors  []error   // All errors encountered (including retries)
}
```

## Error Constants

The package defines several error constants for different failure scenarios:

```go
// ErrServiceAlreadyStarted is returned when Start is called on a Service
// that has already been started. A Service instance must not be started more than once.
var ErrServiceAlreadyStarted = errors.New("service already started")

// ErrNoCheckpointFound is returned by LoadLatestCheckpoint when no checkpoint
// has been saved yet for the requested network.
var ErrNoCheckpointFound = errors.New("no checkpoint found for network")

// ErrNetworkNotRegistered is returned when attempting to operate on an unregistered network.
var ErrNetworkNotRegistered = errors.New("network not registered")
```

## How It Works

### 1. Initialization
The service is created by calling `New`:
```go
service := chainstream.New(networks, options...)
```

### 2. Block Streaming Process

1. **Checkpoint Recovery**: For each network, load the last processed block height
2. **Subscription Setup**: Start streaming from the next block after the checkpoint
3. **Event Processing**: Handle incoming blockchain events:
   - **Success**: Convert to `ObservedBlock` and send for checkpointing
   - **Failure**: Send to retry system (if configured) or failure notifier
4. **Retry Logic**: Attempt to recover failed block fetches
5. **Checkpointing**: Save progress for successful blocks
6. **Output Delivery**: Emit the `ObservedBlock` to the output channel

### 3. Output Stream
The service provides a unified stream of `ObservedBlock` data from all monitored networks.

### 4. Workflow Diagram

Below is a detailed Mermaid diagram illustrating the workflow of the chainstream package, focusing on the process of subscribing to blockchain networks, processing blocks, handling errors, and delivering data to consumers.

```mermaid
graph TD
    subgraph "Service Initialization"
        A["Service begins execution via Start()"] --> B{"Is the service already running?"};
        B -- no --> C["Initialize internal channels (finalOut, preCheckpointCh, etc.)"];
        B -- yes --> D["Return ErrServiceAlreadyStarted error"];
    end

    subgraph "Network Subscription (for each network)"
        C --> E["Load latest checkpoint for the network<br/>(checkpointStorage.LoadLatestCheckpoint)"];
        E --> F["Subscribe to the blockchain client<br/>(blockchain.Subscribe)"];
        F --> G["Receive a stream of BlockchainEvents<br/>(eventsCh)"];
    end

    subgraph "Event Dispatching"
        G --> H["A goroutine processes each event<br/>(dispatchSubscriptionEvents)"];
        H --> I{"Does the event contain an error?"};
        I -- no --> J["On success, send ObservedBlock<br/>to preCheckpointCh"];
        I -- yes --> K["On failure, send BlockDispatchFailure<br/>to errorsCh"];
    end

    subgraph "Error Handling"
        K --> L{"Is a retry mechanism configured?"};
        L -- no --> M["Forward to dispatchFailureCh<br/>for final handling"];
        L -- yes --> N["Forward to retryFailureCh<br/>for reprocessing"];
        N --> O["A goroutine attempts to re-fetch the block<br/>(retryFailedBlockFetches)"];
        O --> P{"Was the block successfully re-fetched?"};
        P -- yes --> J;
        P -- no --> M;
        M --> Q["A goroutine invokes the notifier<br/>(handleDispatchFailures)"];
        Q --> R["DispatchFailureNotifier.NotifyDispatchFailure is executed"];
    end

    subgraph "Final Processing"
        J --> S["A goroutine processes the block<br/>(checkpointAndForward)"];
        S --> T["Persist the new block height<br/>(checkpointStorage.SaveCheckpoint)"];
        T --> V["Send ObservedBlock to the output channel<br/>(finalOut)"];
    end

    subgraph "Service Shutdown"
        W["Service is stopped via Close()"] --> X["Cancel the main context and<br/>close all internal channels"];
    end
```

This diagram provides a detailed overview of the chainstream package workflow:
- **Service Initialization**: The service is started, checks if it's already running, and initializes the necessary channels.
- **Network Subscription**: For each network, it loads the last checkpoint and subscribes to the blockchain to receive a channel of events.
- **Event Dispatching**: Events are dispatched based on whether they contain an error. Successful events go to the processing channel, while errors are routed for handling.
- **Error Handling**: Errors are routed through an optional retry mechanism. If retries fail or are disabled, the error is passed to a user-defined failure notifier.
- **Final Processing**: Successfully fetched blocks are checkpointed and sent to the final output channel.
- **Service Shutdown**: The `Close()` method gracefully shuts down all background processes and closes channels.

## Usage

### Basic Usage

```go
// Assume you have blockchain implementations
networks := map[string]chainstream.Blockchain{
    "ethereum": ethereumClient,
    "polygon":  polygonClient,
}

// Create service (returns ObservedBlock)
service := chainstream.New(networks)

// Start monitoring
ctx := context.Background()
blocksCh, err := service.Start(ctx)
if err != nil {
    return err
}
defer service.Close()

// Process blocks
for block := range blocksCh {
    fmt.Printf("Block from %s: height=%s, txs=%d\n", 
        block.Network, block.Height, len(block.Transactions))
}
```

### Advanced Configuration

```go
// Custom dispatch failure notifier implementation
type customNotifier struct{}

func (c customNotifier) NotifyDispatchFailure(ctx context.Context, failure chainstream.BlockDispatchFailure) error {
    log.Printf("Persistent failure: network=%s height=%s errors=%v", 
        failure.Network, failure.Height, failure.Errors)
    return nil
}

// Standard service with advanced options
service := chainstream.New(networks,
    // Configure retry strategy
    chainstream.WithRetry(retryStrategy),
    
    // Enable checkpoint persistence
    chainstream.WithCheckpointStorage(storage),
    
    // Custom failure notifier
    chainstream.WithDispatchFailureNotifier(customNotifier{}),
)
```

## Configuration Options

### WithRetry
Configure retry logic for transient failures:
```go
chainstream.WithRetry(retryStrategy)
```

### WithCheckpointStorage
Enable checkpoint persistence to resume from last processed block:
```go
chainstream.WithCheckpointStorage(storage)
```

### WithDispatchFailureNotifier
Set custom notifier for unrecoverable failures:
```go
chainstream.WithDispatchFailureNotifier(notifier)
```

## Error Handling

### Error Flow
1. **Transient Errors**: Sent to retry system (if configured)
2. **Persistent Errors**: Sent to dispatch failure notifier
3. **Critical Errors**: Service startup failures returned immediately

### Default Implementations

The package provides default implementations for optional components:

#### Default Checkpoint Storage
```go
// nopCheckpoint is a no-op implementation that disables checkpointing
type nopCheckpoint struct{}
```

#### Default Dispatch Failure Notifier
```go
// logDispatchFailureNotifier logs failures using the application's logger
type logDispatchFailureNotifier struct{}
```

## Internal Configuration

The service uses buffered channels with predefined sizes for optimal performance:

```go
const (
    dispatchFailureChannelBufferSize = 5  // Buffer size for dispatch failure events
    retryFailureChannelBufferSize    = 5  // Buffer size for failures retried by retry logic
    observedBlockChannelBufferSize   = 10 // Buffer size for final successfully observed blocks
)
```

## Features

### Resilience
- **Retry Logic**: Configurable retry strategies for transient failures
- **Error Isolation**: Network failures don't affect other networks
- **Graceful Degradation**: Continue processing other networks on partial failures

### Scalability
- **Concurrent Processing**: Each network runs independently
- **Buffered Channels**: Configurable buffer sizes for optimal throughput
- **Resource Management**: Proper cleanup and resource management
- **Asynchronous Processing**: Non-blocking data transformation pipeline

### Reliability
- **Checkpoint System**: Resume from last processed block after restarts
- **No Data Loss**: Failed blocks are tracked and retried
- **Context Cancellation**: Proper cancellation handling throughout

## Thread Safety

The service is designed to be thread-safe:
- **Single Start**: Service can only be started once (returns `ErrServiceAlreadyStarted`)
- **Concurrent Access**: Safe to call `Close()` from any goroutine
- **Channel Safety**: All internal channels are properly synchronized

## Dependencies

The package has minimal external dependencies:
- **Blockchain Implementations**: Must implement the `Blockchain` interface
- **Retry Strategy**: Optional, must implement `retry.Retry` interface
- **Checkpoint Storage**: Optional, must implement `CheckpointStorage` interface
- **Dispatch Failure Notifier**: Optional, must implement `DispatchFailureNotifier` interface
- **Context**: Standard Go context for cancellation and timeouts

## Integration

This package is designed to be used within the larger blockwatch project as the core blockchain monitoring component. It provides a clean abstraction over multiple blockchain networks while handling the complexities of network failures and state management.

The package expects blockchain implementations to be provided by other components in the project (e.g., `internal/infra/blockchain/ethereum`) and can optionally integrate with storage backends for checkpoint persistence.
