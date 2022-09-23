package main

import (
	"testing"
)

func TestEnv(t *testing.T) {
	// backend server address/port flag
	// frontend address/port flag
	// timeout - read, write, idle
	//

	if err := loadConfig(); err != nil {
		t.Fatal(err)
	}
}
