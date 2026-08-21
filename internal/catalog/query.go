package catalog

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/database"
	"github.com/Blustak/go-factorioHelper/internal/datatypes"
)

func (c *Catalog) Query(cfg *config.State, category *string, query string) error {
	if cfg == nil {
		return fmt.Errorf("nil config")
	}
	if cfg.Writer == nil {
		return fmt.Errorf("nil writer")
	}
	if category == nil {
		if err := c.queryAll(cfg, query); err != nil {
			return err
		}
		return cfg.Writer.Flush()
	}
	switch cat := *category; cat {
	case "items":
		if err := c.queryItems(cfg, query); err != nil {
			return err
		}
	case "recipes":
		if err := c.queryRecipes(cfg, query); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown category: %s", cat)
	}
	return cfg.Writer.Flush()
}

func (c *Catalog) queryAll(cfg *config.State, query string) error {
	for _, f := range []func(*config.State, string) error{
		c.queryItems,
		c.queryRecipes,
	} {
		if err := f(cfg, query); err != nil {
			return err
		}
	}
	return nil
}

func (c *Catalog) queryItems(cfg *config.State, query string) error {
	output, err := cfg.DB.SearchItemsByName(cfg.CTX, likePattern(query))
	if err != nil {
		cfg.Log.Error("failed to query items", "error", err, "query", query)
		return err
	}
	for _, s := range formatItemQueryResults(output) {
		if _, err := cfg.Writer.WriteString(s); err != nil {
			return err
		}
	}
	return nil
}

func (c *Catalog) queryRecipes(cfg *config.State, query string) error {
	output, err := cfg.DB.SearchRecipesByName(cfg.CTX, likePattern(query))
	if err != nil {
		cfg.Log.Error("failed to query recipes", "error", err, "query", query)
		return err
	}
	lines, err := formatRecipeQueryResults(output)
	if err != nil {
		return err
	}
	for _, s := range lines {
		if _, err := cfg.Writer.WriteString(s); err != nil {
			return err
		}
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

	return fmt.Sprintf("[item] Name: %s, Stack Size: %d, Fuel Value: %.2fJ\n", name, stackSize, fuelValue)
}

func formatRecipeQueryResults(query []database.SearchRecipesByNameRow) ([]string, error) {
	var s []string
	for _, q := range query {
		line, err := formatRecipeQueryResult(q)
		if err != nil {
			return nil, err
		}
		s = append(s, line)
	}
	return s, nil
}

func formatRecipeQueryResult(query database.SearchRecipesByNameRow) (string, error) {
	name := query.Name.String
	rec, err := datatypes.RecipeFromDB(name, "recipe", sql.NullString{}, database.Recipe{
		EnergyRequired: query.EnergyRequired,
		Category:       query.Category,
		MainProduct:    query.MainProduct,
		Ingredient:     query.Ingredient,
		Results:        query.Results,
	})
	if err != nil {
		return "", err
	}

	category := ""
	if rec.Category != nil {
		category = *rec.Category
	}
	energy := 0.0
	if rec.EnergyRequired != nil {
		energy = *rec.EnergyRequired
	}

	return fmt.Sprintf(
		"[recipe] Name: %s, Category: %s, Energy: %.2fs, Ingredients: %s, Results: %s\n",
		name,
		category,
		energy,
		formatIngredients(rec.Ingredients),
		formatProducts(rec.Results),
	), nil
}

func formatIngredients(ings []datatypes.Ingredient) string {
	if len(ings) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(ings))
	for _, in := range ings {
		parts = append(parts, formatNamedAmount(in.Name, expectedIngredientAmount(in)))
	}
	return strings.Join(parts, ", ")
}

func formatProducts(products []datatypes.Product) string {
	if len(products) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(products))
	for _, p := range products {
		parts = append(parts, formatNamedAmount(p.Name, expectedProductAmount(p)))
	}
	return strings.Join(parts, ", ")
}

func formatNamedAmount(name string, amount float64) string {
	if amount == float64(int64(amount)) {
		return fmt.Sprintf("%s x%d", name, int64(amount))
	}
	return fmt.Sprintf("%s x%.2f", name, amount)
}

func likePattern(query string) string {
	return "%" + query + "%"
}

func fromNullInt64(i sql.NullInt64) (r int64) {
	if i.Valid {
		r = i.Int64
	}
	return r
}

func fromNullFloat64(f sql.NullFloat64) (r float64) {
	if f.Valid {
		r = f.Float64
	}
	return r
}
