package config

type Mode int

const (
	Mysql Mode = iota + 1
	Postgres
	ClickHouse
)

type Config struct {
	Mode                   Mode   `json:"mode"`
	Trace                  bool   `json:"trace,default=false"`
	DSN                    string `json:"dsn,optional"`
	Debug                  bool   `json:"debug,default=false"`
	MaxIdleConn            int    `json:"max_idle_conn"`
	MaxOpenConn            int    `json:"max_open_conn"`
	MaxLifetime            int    `json:"max_lifetime"`
	PrepareStmt            bool   `json:"prepare_stmt"`
	SkipDefaultTransaction bool   `json:"skip_default_transaction"`
}
