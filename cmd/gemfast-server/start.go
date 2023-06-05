package cmd

import (
	"github.com/gemfast/server/internal/api"
	"github.com/gemfast/server/internal/cve"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/filter"
	"github.com/gemfast/server/internal/indexer"
	"github.com/gemfast/server/internal/license"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the gemfast rubygems server",
	Long:  "Reads in the gemfast config file and starts the gemfast rubygems server",
	Run: func(cmd *cobra.Command, args []string) {
		start()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func start() {
	err := license.ValidateLicenseKey()
	check(err)
	log.Info().Msg("starting services")
	cve.InitRubyAdvisoryDB()
	err = db.Connect()
	check(err)
	defer db.BoltDB.Close()
	err = indexer.InitIndexer()
	check(err)
	err = indexer.Get().GenerateIndex()
	check(err)
	err = filter.InitFilter()
	check(err)
	err = api.Run()
	check(err)
}
