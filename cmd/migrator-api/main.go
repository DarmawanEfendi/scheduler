package main

import (
	"os"

	"github.com/DarmawanEfendi/scheduler/cmd/migrator-api/server"
)

func main() {
	os.Exit(server.Run())
}
