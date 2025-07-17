package messaging

type RabbitMQ struct {
	URI string `validate:"required"`
}

func (r RabbitMQ) IsActivated() bool {
	return r.URI != ""
}
