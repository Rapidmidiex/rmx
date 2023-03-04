package postgres_test

import (
	"context"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	db "github.com/rapidmidiex/rmx/internal/jams/postgres/sqlc"
	"github.com/stretchr/testify/require"
)

func TestCreateJam(t *testing.T) {
	jamName := gofakeit.NounAbstract()
	want := db.Jam{
		Name: jamName,
		Bpm:  90,
		// Defaults
		Capacity: 5,
	}
	arg := db.CreateJamParams{
		Name:     want.Name,
		Bpm:      want.Bpm,
		Capacity: want.Capacity,
	}
	got, err := testQueries.CreateJam(context.Background(), &arg)
	require.NoError(t, err)

	require.NotEmpty(t, got.ID, "ID should have a value")
	require.Equal(t, want.Name, got.Name)
	require.Equal(t, want.Bpm, got.Bpm)
	require.Equal(t, want.Capacity, got.Capacity)
}
