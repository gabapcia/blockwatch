package messaging

type Redis struct {
	Address  string `validate:"required"`
	Username string
	Password string
	DB       int
}

func (r Redis) IsActivated() bool {
	return r.Address != ""
}
