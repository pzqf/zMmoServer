package models

import (
	"time"
)

type PlayerPet struct {
	PetID      int64     `db:"pet_id" bson:"pet_id"`
	PlayerID   int64     `db:"player_id" bson:"player_id"`
	PetConfigID int32    `db:"pet_config_id" bson:"pet_config_id"`
	Name       string    `db:"name" bson:"name"`
	Level      int32     `db:"level" bson:"level"`
	Exp        int64     `db:"exp" bson:"exp"`
	HP         int32     `db:"hp" bson:"hp"`
	MaxHP      int32     `db:"max_hp" bson:"max_hp"`
	Attack     int32     `db:"attack" bson:"attack"`
	Defense    int32     `db:"defense" bson:"defense"`
	Skills     string    `db:"skills" bson:"skills"`
	IsActive   int32     `db:"is_active" bson:"is_active"`
	CreatedAt  time.Time `db:"created_at" bson:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" bson:"updated_at"`
}

func (PlayerPet) TableName() string {
	return "`player_pets`"
}
