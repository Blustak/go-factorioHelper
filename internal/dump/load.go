package dump

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/datatypes"
)

var knownTypes = map[string]struct{}{
	"item":               {},
	"fluid":              {},
	"resource":           {},
	"recipe":             {},
	"assembling-machine": {},
	"furnace":            {},
	"mining-drill":       {},
	"offshore-pump":      {},
	"pump":               {},
	"boiler":             {},
	"generator":          {},
}

type Stats struct {
	Items             int
	Fluids            int
	Resources         int
	Recipes           int
	AssemblyMachines  int
	Furnaces          int
	ResourceProducers int
	Boilers           int
	Generators        int
	Skipped           int
}

func Load(cfg *config.State, path string) (Stats, error) {
	var stats Stats
	data, err := os.ReadFile(path)
	if err != nil {
		return stats, err
	}

	var dumpFile map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &dumpFile); err != nil {
		return stats, fmt.Errorf("parse dump: %w", err)
	}

	for kind, objects := range dumpFile {
		if _, ok := knownTypes[kind]; !ok {
			stats.Skipped += len(objects)
		}
	}

	err = cfg.RunInTx(func(txCfg *config.State) error {
		n, err := loadMap(dumpFile["item"], "item", func(v *datatypes.Item) error {
			_, err := v.AddToDB(txCfg)
			return err
		})
		if err != nil {
			return err
		}
		stats.Items = n

		n, err = loadMap(dumpFile["fluid"], "fluid", func(v *datatypes.Fluid) error {
			_, err := v.AddToDB(txCfg)
			return err
		})
		if err != nil {
			return err
		}
		stats.Fluids = n

		n, err = loadMap(dumpFile["resource"], "resource", func(v *datatypes.Resource) error {
			_, err := v.AddToDB(txCfg)
			return err
		})
		if err != nil {
			return err
		}
		stats.Resources = n

		n, err = loadMap(dumpFile["recipe"], "recipe", func(v *datatypes.Recipe) error {
			_, err := v.AddToDB(txCfg)
			return err
		})
		if err != nil {
			return err
		}
		stats.Recipes = n

		n, err = loadMap(dumpFile["assembling-machine"], "assembling-machine", func(v *datatypes.AssemblyMachine) error {
			_, err := v.AddToDB(txCfg)
			return err
		})
		if err != nil {
			return err
		}
		stats.AssemblyMachines = n

		n, err = loadMap(dumpFile["furnace"], "furnace", func(v *datatypes.AssemblyMachine) error {
			_, err := v.AddToDB(txCfg)
			return err
		})
		if err != nil {
			return err
		}
		stats.Furnaces = n

		producers := 0
		for _, kind := range []string{"mining-drill", "offshore-pump", "pump"} {
			n, err = loadMap(dumpFile[kind], kind, func(v *datatypes.ResourceProducer) error {
				_, err := v.AddToDB(txCfg)
				return err
			})
			if err != nil {
				return err
			}
			producers += n
		}
		stats.ResourceProducers = producers

		n, err = loadMap(dumpFile["boiler"], "boiler", func(v *datatypes.Boiler) error {
			_, err := v.AddToDB(txCfg)
			return err
		})
		if err != nil {
			return err
		}
		stats.Boilers = n

		n, err = loadMap(dumpFile["generator"], "generator", func(v *datatypes.Generator) error {
			_, err := v.AddToDB(txCfg)
			return err
		})
		if err != nil {
			return err
		}
		stats.Generators = n
		return nil
	})
	if err != nil {
		return Stats{}, err
	}
	return stats, nil
}

func loadMap[T any](objects map[string]json.RawMessage, kind string, add func(*T) error) (int, error) {
	n := 0
	for name, raw := range objects {
		var v T
		if err := json.Unmarshal(raw, &v); err != nil {
			return n, fmt.Errorf("%s %q: %w", kind, name, err)
		}
		if err := add(&v); err != nil {
			return n, fmt.Errorf("%s %q: %w", kind, name, err)
		}
		n++
	}
	return n, nil
}
