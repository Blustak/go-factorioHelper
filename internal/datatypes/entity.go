package datatypes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/database"
)

type DatadumpDatatype interface {
	Entity | Item | Fluid | Resource | Recipe | AssemblyMachine | ResourceProducer
}

type DatatypeEntry[U interface {
	database.Entity |
		database.Item |
		database.Fluid |
		database.Resource |
		database.Recipe |
		database.AssemblyMachine |
		database.ResourceProducer
}] interface {
	json.Unmarshaler
	AddToDB(cfg *config.State) (U, error)
	GetFromDB(cfg *config.State) (U, error)
	GetEntityID(cfg *config.State) (int64, error)
	Unwrap() (U, error)
}

type DatatypeNilError[T DatadumpDatatype] string

func (err DatatypeNilError[T]) Error() string {
	return fmt.Sprintf("datatype nil error; %s", string(err))
}

type Entity struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	EntityOrder *string `json:"order"`
	entEntry    *database.Entity
}

func (e *Entity) AddToDB(cfg *config.State) (database.Entity, error) {
	var ent database.Entity
	if e == nil {
		return ent, DatatypeNilError[Entity]("Nil entity")
	}
	if e.Type == "" {
		return ent, fmt.Errorf("entity type is required")
	}

	existing, err := cfg.DB.GetEntityByName(cfg.CTX, database.GetEntityByNameParams{
		Name:          e.Name,
		PrototypeType: e.Type,
	})
	if err == nil {
		if e.EntityOrder != nil && !existing.EntityOrder.Valid {
			existing, err = cfg.DB.UpdateEntityOrderByID(cfg.CTX, database.UpdateEntityOrderByIDParams{
				Order: sql.NullString{String: *e.EntityOrder, Valid: true},
				ID:    existing.ID,
			})
			if err != nil {
				return ent, err
			}
		}
		e.entEntry = &existing
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return ent, err
	}

	entityParams := database.AddEntityParams{
		Name:          e.Name,
		PrototypeType: e.Type,
		EntityOrder:   sql.NullString{Valid: false},
	}
	if e.EntityOrder != nil {
		entityParams.EntityOrder = sql.NullString{String: *e.EntityOrder, Valid: true}
	}

	ent, err = cfg.DB.AddEntity(cfg.CTX, entityParams)
	if err != nil {
		return ent, err
	}
	e.entEntry = &ent
	return ent, nil
}

func (e *Entity) GetFromDB(cfg *config.State) (database.Entity, error) {
	var ent database.Entity
	if e == nil {
		return ent, DatatypeNilError[Entity]("Nil entity")
	}

	ent, err := cfg.DB.GetEntityByName(cfg.CTX, database.GetEntityByNameParams{
		Name:          e.Name,
		PrototypeType: e.Type,
	})
	if err != nil {
		return ent, err
	}
	if e.entEntry == nil {
		return ent, fmt.Errorf("entEntry not set")
	}
	if *e.entEntry != ent {
		cfg.Log.Error("Entity datatype does not match data in database", "entity", *e.entEntry, "database entity", ent)
		return ent, fmt.Errorf("Datatype does not match database entry")
	}
	return ent, nil
}

func (e *Entity) Unwrap() (database.Entity, error) {
	if e == nil {
		return database.Entity{}, DatatypeNilError[Entity]("Nil entity")
	}
	if e.entEntry == nil {
		return database.Entity{}, fmt.Errorf("entEntry not set")
	}
	return *e.entEntry, nil
}

func (e *Entity) UnmarshalJSON(bytes []byte) error {
	type alias Entity
	var a alias
	if err := json.Unmarshal(bytes, &a); err != nil {
		return err
	}
	*e = Entity(a)
	return nil
}

var (
	_ DatatypeEntry[database.Entity]           = (*Entity)(nil)
	_ DatatypeEntry[database.Item]             = (*Item)(nil)
	_ DatatypeEntry[database.Fluid]            = (*Fluid)(nil)
	_ DatatypeEntry[database.Resource]         = (*Resource)(nil)
	_ DatatypeEntry[database.Recipe]           = (*Recipe)(nil)
	_ DatatypeEntry[database.AssemblyMachine]  = (*AssemblyMachine)(nil)
	_ DatatypeEntry[database.ResourceProducer] = (*ResourceProducer)(nil)
)

func (e *Entity) GetEntityID(cfg *config.State) (int64, error) {
	var id int64
	if e == nil {
		return id, DatatypeNilError[Entity]("Cannot get id of nil entity")
	}

	if e.entEntry == nil {
		return id, fmt.Errorf("Tried to get id of entity with no underlying entity data")
	}

	return e.entEntry.ID, nil
}

func ensureEntity(cfg *config.State, name, prototypeType string) (database.Entity, error) {
	e := &Entity{Name: name, Type: prototypeType}
	return e.AddToDB(cfg)
}

func stubEntityID(cfg *config.State, name *string, prototypeType string) (sql.NullInt64, error) {
	if name == nil || *name == "" {
		return sql.NullInt64{}, nil
	}
	ent, err := ensureEntity(cfg, *name, prototypeType)
	if err != nil {
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: ent.ID, Valid: true}, nil
}

func mismatchError(cfg *config.State, kind string, cached, got any) error {
	cfg.Log.Error(kind+" datatype does not match data in database", "cached", cached, "database", got)
	return fmt.Errorf("Datatype does not match database entry")
}

func rowsEqual(a, b any) bool {
	return reflect.DeepEqual(a, b)
}
