package storage

import (
	"database/sql"

	_ "github.com/lib/pq" // postgres driver
	"sync"
	"github.com/homeserver-today/react-sdk-config-server/config"
	"github.com/sirupsen/logrus"
	"github.com/DavidHuie/gomigrate"
	"context"
	"github.com/homeserver-today/react-sdk-config-server/models"
	"encoding/json"
)

const selectConfig = "SELECT config FROM configs WHERE hostname = $1;"
const upsertConfig = "INSERT INTO configs (hostname, config) VALUES ($1, $2) ON CONFLICT (hostname) DO UPDATE SET config = $2;"
const deleteConfig = "DELETE FROM configs WHERE hostname = $1;"
const selectGlobs = "SELECT hostname FROM configs WHERE hostname LIKE '%*%';"

type Database struct {
	db         *sql.DB
	statements statements
}

type statements struct {
	selectConfig *sql.Stmt
	upsertConfig *sql.Stmt
	deleteConfig *sql.Stmt
	selectGlobs  *sql.Stmt
}

var dbInstance *Database
var singletonDbLock = &sync.Once{}

func GetDatabase() (*Database) {
	if dbInstance == nil {
		singletonDbLock.Do(func() {
			err := OpenDatabase(config.Get().Database.Postgres)
			if err != nil {
				panic(err)
			}
		})
	}
	return dbInstance
}

func OpenDatabase(connectionString string) (error) {
	d := &Database{}
	var err error

	if d.db, err = sql.Open("postgres", connectionString); err != nil {
		return err
	}

	// Make sure the database is how we want it
	migrator, err := gomigrate.NewMigratorWithLogger(d.db, gomigrate.Postgres{}, config.Runtime.MigrationsPath, logrus.StandardLogger())
	if err != nil {
		return err
	}
	err = migrator.Migrate()
	if err != nil {
		return err
	}

	// Prepare the general statements
	if d.statements.selectConfig, err = d.db.Prepare(selectConfig); err != nil {
		return err
	}
	if d.statements.upsertConfig, err = d.db.Prepare(upsertConfig); err != nil {
		return err
	}
	if d.statements.deleteConfig, err = d.db.Prepare(deleteConfig); err != nil {
		return err
	}
	if d.statements.selectGlobs, err = d.db.Prepare(selectGlobs); err != nil {
		return err
	}

	dbInstance = d
	return nil
}

func (d *Database) GetConfig(ctx context.Context, domain string) (models.ReactConfig, error) {
	configStr := ""
	err := d.statements.selectConfig.QueryRowContext(ctx, domain).Scan(&configStr)
	if err != nil {
		return nil, err
	}

	reactConfig := models.ReactConfig{}
	err = json.Unmarshal([]byte(configStr), &reactConfig)
	return reactConfig, err
}

func (d *Database) UpsertConfig(ctx context.Context, domain string, config models.ReactConfig) (error) {
	configStr, err := json.Marshal(config)
	if err != nil {
		return err
	}

	_, err = d.statements.upsertConfig.ExecContext(ctx, domain, string(configStr))
	return err
}

func (d *Database) DeleteConfig(ctx context.Context, domain string) (error) {
	_, err := d.statements.deleteConfig.ExecContext(ctx, domain)
	return err
}

func (d *Database) ListGlobs(ctx context.Context) ([]string, error) {
	rows, err := d.statements.selectGlobs.QueryContext(ctx)
	if err == sql.ErrNoRows {
		return make([]string, 0), nil
	} else if err != nil {
		return nil, err
	}

	results := make([]string, 0)
	var scanString string
	for rows.Next() {
		rows.Scan(&scanString)
		results = append(results, scanString)
	}

	return results, err
}
