package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/dump"
	"github.com/joho/godotenv"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s load <dump.json>\n", os.Args[0])
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	_ = godotenv.Load()

	if len(args) < 1 {
		usage()
		return 2
	}

	switch args[0] {
	case "load":
		if len(args) != 2 {
			usage()
			fmt.Fprintln(os.Stderr, "load requires a dump file path")
			return 2
		}
		dbString := os.Getenv("GOOSE_DBSTRING")
		if dbString == "" {
			fmt.Fprintln(os.Stderr, "GOOSE_DBSTRING is not set")
			return 1
		}
		ctx := context.Background()
		cfg, err := config.Open(ctx, dbString)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer cfg.Close()

		stats, err := dump.Load(cfg, args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		cfg.Log.Info("loaded dump",
			"items", stats.Items,
			"fluids", stats.Fluids,
			"resources", stats.Resources,
			"recipes", stats.Recipes,
			"assembling_machines", stats.AssemblyMachines,
			"skipped", stats.Skipped,
		)
		return 0
	default:
		usage()
		fmt.Fprintf(os.Stderr, "unknown command %q\n", args[0])
		return 2
	}
}
