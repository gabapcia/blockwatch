package storage

type Redis struct {
	Address  string `validate:"required"`
	Username string
	Password string
	DB       int
}
