package datatypes

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Blustak/go-factorioHelper/internal/config"
	"github.com/Blustak/go-factorioHelper/internal/database"
)

type DatadumpDatatype interface {
	Entity
}

type DatatypeEntry[U interface {
	database.Entity |
		database.Item
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
	EntityOrder *string `json:"order"`
	entEntry    *database.Entity
}

func (e *Entity) AddToDB(cfg *config.State) (database.Entity, error) {
	var ent database.Entity
	if e == nil {
		return ent, DatatypeNilError[Entity]("Nil entity")
	}

	entityParams := database.AddEntityParams{
		Name:        e.Name,
		EntityOrder: sql.NullString{Valid: false},
	}

	if e.EntityOrder != nil {
		entityParams.EntityOrder = sql.NullString{String: *e.EntityOrder, Valid: true}
	}

	var err error
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

	ent, err := cfg.DB.GetEntityByName(cfg.CTX, e.Name)
	if err != nil {
		return ent, err
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
