package storage

const (
	EngineRedis      = "REDIS"
	EnginePostgreSQL = "POSTGRESQL"
)

type Engines struct {
	Redis    Redis      `validate:"omitempty"`
	Postgres PostgreSQL `validate:"omitempty"`
}

type InlineConfig struct {
	Redis    Redis      `validate:"required_alone"`
	Postgres PostgreSQL `validate:"required_alone"`
}

type Picker struct {
	Engine string       `validate:"omitempty,oneof=REDIS POSTGRESQL"`
	Config InlineConfig `validate:"required_without=Engine,excluded_with=Engine"`
}
