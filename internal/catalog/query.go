package catalog

import (
	"database/sql"
	"fmt"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/database"
)

func (c *Catalog) Query(cfg *config.State, category *string, query string) error {
  if cfg == nil {
    return fmt.Errorf("nil config")
  }
  if category == nil {
    return c.queryAll(cfg, query)
  }
  switch cat := *category; cat {
  case "items":
    return c.queryItems(cfg, query)
  default:
    return fmt.Errorf("unknown category: %s", cat)
  }
}

func (c *Catalog) queryAll(cfg *config.State, query string) error {
  for _, f := range []func(*config.State, string) error{
    c.queryItems,
  } {
    if err := f(cfg, query); err != nil {
      return err
    }
  }
  return nil
}

func (c *Catalog) queryItems(cfg *config.State, query string) error{
  output, err := cfg.DB.SearchItemsByName(cfg.CTX,"%"+query+"%")
  if err != nil {
    return err
  }
  for _, s := range formatItemQueryResults(output) {
    cfg.Writer.WriteString(s)
  }
  return nil
}

func formatItemQueryResults(query []database.SearchItemsByNameRow) []string {
  var s []string
  for _, q := range query {
    s = append(s, formatItemQueryResult(q))
  }
  return s
}

func formatItemQueryResult(query database.SearchItemsByNameRow) string {
  name := query.Name.String
  stackSize := fromNullInt64(query.StackSize)
  fuelValue := fromNullFloat64(query.FuelValue)

  return fmt.Sprintf("Name: %s, Stack Size: %d, Fuel Value: %.2fJ", name,stackSize,fuelValue)
}


func fromNullInt64(i sql.NullInt64) (r int64) {
  if i.Valid {
    r = i.Int64
  }
  return i.Int64
}

func fromNullFloat64(f sql.NullFloat64) (r float64) {
  if f.Valid{
    r = f.Float64
  }
  return r
}
