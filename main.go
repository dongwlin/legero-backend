package main

import (
	"github.com/rs/zerolog/log"

	"github.com/dongwlin/legero-backend/internal/wire"
)

func main() {
	app, err := wire.InitApp()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("failed to initialize application")
	}

	app.Run()
}
