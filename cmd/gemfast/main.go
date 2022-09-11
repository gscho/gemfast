package main

import (
	"fmt"

	"github.com/gscho/gemfast/internal/api"
	"github.com/gscho/gemfast/internal/db"
	"github.com/gscho/gemfast/internal/indexer"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func init() {
	viper.SetEnvPrefix("GEMFAST")
	viper.SetDefault("dir", "/var/gemfast")
	viper.SetDefault("gem_dir", fmt.Sprintf("%s/gems", viper.Get("dir")))
	viper.SetDefault("db_dir", "db")
	viper.SetDefault("auth", "local")
	viper.AutomaticEnv()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func main() {
	log.Info().Msg("starting services")
	err := db.Connect()
	if err != nil {
		panic(err)
	}
	defer db.BoltDB.Close()
	err = indexer.InitIndexer()
	if err != nil {
		panic(err)
	}
	// indexer.Get().GenerateIndex()
	err = api.Run()
	if err != nil {
		panic(err)
	}
}
