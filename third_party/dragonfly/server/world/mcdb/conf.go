package mcdb

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb/leveldat"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
)

type Config struct {
	Log *slog.Logger

	LDBOptions *opt.Options

	Blocks world.BlockRegistry
}

func (conf Config) Open(dir string) (*DB, error) {
	if conf.Log == nil {
		conf.Log = slog.Default()
	}
	conf.Log = conf.Log.With("provider", "mcdb")
	if conf.LDBOptions == nil {
		conf.LDBOptions = new(opt.Options)
	}
	if conf.LDBOptions.BlockSize == 0 {
		conf.LDBOptions.BlockSize = 16 * opt.KiB
	}

	_ = os.MkdirAll(filepath.Join(dir, "db"), 0777)

	db := &DB{conf: conf, dir: dir, ldat: &leveldat.Data{}}
	db.SetBlockRegistry(conf.Blocks)
	if _, err := os.Stat(filepath.Join(dir, "level.dat")); os.IsNotExist(err) {

		db.ldat.FillDefault()
	} else {
		ldat, err := leveldat.ReadFile(filepath.Join(dir, "level.dat"))
		if err != nil {
			return nil, fmt.Errorf("open db: read level.dat: %w", err)
		}
		ver := ldat.Ver()
		if ver != leveldat.Version && ver >= 10 {
			return nil, fmt.Errorf("open db: level.dat version %v is unsupported", ver)
		}
		if err = ldat.Unmarshal(db.ldat); err != nil {
			return nil, fmt.Errorf("open db: unmarshal level.dat: %w", err)
		}
	}
	db.set = db.ldat.Settings()
	ldb, err := leveldb.OpenFile(filepath.Join(dir, "db"), conf.LDBOptions)
	if err != nil {
		return nil, fmt.Errorf("open db: leveldb: %w", err)
	}
	db.ldb = ldb
	return db, nil
}
