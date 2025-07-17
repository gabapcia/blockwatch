package storage

type PostgreSQL struct {
	URI string `validate:"required"`
}

func (p PostgreSQL) IsActivated() bool {
	return p.URI != ""
}
