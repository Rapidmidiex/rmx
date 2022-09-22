package main

import (
	"testing"

	"github.com/spf13/viper"
)

func TestEnv(t *testing.T) {
	viper.SetConfigFile(".env")
}
