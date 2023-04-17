package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/gorilla/mux"
	"github.com/urfave/cli/v2"
)

const (
	defaultDataDir = "/tmp/victor"
	defaultHost    = "localhost:6723"
)

func main() {
	app := &cli.App{
		Name:  "Victor",
		Usage: "A simple vector database",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "data-dir",
				Value:   defaultDataDir,
				Usage:   "Directory to store the data",
				EnvVars: []string{"DATA_DIR"},
			},
			&cli.StringFlag{
				Name:    "host",
				Value:   defaultHost,
				Usage:   "Host and port to listen on",
				EnvVars: []string{"HOST"},
			},
		},
		Action: func(c *cli.Context) error {
			dataDir := c.String("data-dir")
			host := c.String("host")

			opts := badger.DefaultOptions(dataDir)
			opts.Logger = nil
			var err error
			db, err := badger.Open(opts)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Database opened at %s.\n", dataDir)
			defer db.Close()
			index := NewVectorIndex(50)
			log.Println("Restoring index...")
			t0 := time.Now()
			index.restoreIndex(db)
			log.Printf("Index restored in %v. %d items loaded.\n", time.Since(t0), index.Len())

			r := mux.NewRouter()
			s := NewServer(db, index)
			s.RegisterRoutes(r)

			// print memory usage every 1 minute
			go func() {
				ticker := time.NewTicker(time.Minute)
				for range ticker.C {
					var m runtime.MemStats
					runtime.ReadMemStats(&m)
					log.Printf("Memory usage:")
					log.Printf("\tAlloc = %v MiB", bToMb(m.Alloc))
					log.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
					log.Printf("\tSys = %v MiB", bToMb(m.Sys))
				}
			}()

			log.Printf("Starting server on %s...\n", host)
			if err := http.ListenAndServe(host, r); err != nil {
				return fmt.Errorf("failed to start server: %w", err)
			}

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
