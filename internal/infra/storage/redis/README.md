# Redis Storage

This package provides a Redis-backed implementation of the storage and coordination interfaces required by the Blockwatch system.

## Technologies Used

- **[redis/go-redis](https://github.com/redis/go-redis):** The official Go client for Redis.
- **[testcontainers-go](https://github.com/testcontainers/testcontainers-go):** Used for integration testing to spin up a Redis container, ensuring that tests run in a clean and isolated environment.

## How It Works

The core of the package is the `Client` interface, which aggregates several interfaces from other parts of the application:

- `chainstream.CheckpointStorage`: For persisting chainstream checkpoints.
- `walletregistry.WalletStorage`: For managing wallet registry data.
- `walletwatch.WalletStorage`: For tracking active wallets.
- `walletwatch.IdempotencyGuard`: For preventing duplicate event processing.

The `client` struct is the concrete implementation of this interface, using a `redis.Client` to interact with the Redis server. The `New` function is responsible for creating and initializing a new client, including establishing the connection and verifying it with a `PING` command.

## Unit Test Strategy

The testing strategy for this package relies on integration tests against a real Redis instance. This is achieved using `testcontainers-go`, which automatically manages the lifecycle of a Redis Docker container.

A helper function, `setupRedisContainer`, is responsible for:
1. Starting a Redis container.
2. Retrieving the connection string.
3. Creating a new `client` instance connected to the container.
4. Providing a cleanup function to terminate the container and close the client connection after the test completes.

This approach ensures that the tests are reliable and validate the client's behavior against a live Redis server without requiring a manually configured Redis instance.
