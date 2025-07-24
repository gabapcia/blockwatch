# RabbitMQ Messaging

This package provides a RabbitMQ-backed implementation of the messaging and notification interfaces required by the Blockwatch system.

## Technologies Used

- **[rabbitmq/amqp091-go](https://github.com/rabbitmq/amqp091-go):** The official Go client for RabbitMQ.
- **[testcontainers-go](https://github.com/testcontainers/testcontainers-go):** Used for integration testing to spin up a RabbitMQ container, ensuring that tests run in a clean and isolated environment.

## How It Works

The core of the package is the `Client` interface, which provides methods to create domain-specific adapters for messaging capabilities:

- `chainstream.DispatchFailureNotifier`: For publishing notifications about chainstream dispatch failures.
- `walletwatch.TransactionNotifier`: For publishing notifications about new transactions for watched wallets.

The `client` struct is the concrete implementation of this interface, using an `amqp.Channel` to publish messages to a RabbitMQ server. The `New` function is responsible for creating and initializing a new client, including establishing the connection and opening a channel.

## Unit Test Strategy

The testing strategy for this package relies on integration tests against a real RabbitMQ instance. This is achieved using `testcontainers-go`, which automatically manages the lifecycle of a RabbitMQ Docker container.

A helper function, `setupRabbitMQContainer` (defined in the test files), is responsible for:
1. Starting a RabbitMQ container.
2. Retrieving the connection URI.
3. Creating a new `client` instance connected to the container.
4. Providing a cleanup function to terminate the container and close the client connection after the test completes.

This approach ensures that the tests are reliable and validate the client's behavior against a live RabbitMQ server without requiring a manually configured instance.
