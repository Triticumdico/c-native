package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

var logger TransactionLogger

func initializeTransactionLog() error {
	var err error

	logger, err = NewFileTransactionLogger("/tmp/transactions.log")
	if err != nil {
		return fmt.Errorf("failed to create transaction logger: %w", err)
	}

	events, errors := logger.ReadEvents()
	count, ok, e := 0, true, Event{}

	for ok && err == nil {
		select {
		case err, ok = <-errors:

		case e, ok = <-events:
			switch e.EventType {
			case EventDelete: // Got a DELETE event!
				err = Delete(e.Key)
				count++
			case EventPut: // Got a PUT event!
				err = Put(e.Key, e.Value)
				count++
			}
		}
	}

	log.Printf("%d events replayed\n", count)

	logger.Run()

	return err
}

func main() {

	// Initializes the transaction log and loads existing data, if any.
	// Blocks until all data is read.
	err := initializeTransactionLog()
	if err != nil {
		panic(err)
	}

	// Create a new mux router
	r := mux.NewRouter()

	r.HandleFunc("/v1/{key}", keyValueGetHandler).Methods("GET")
	r.HandleFunc("/v1/{key}", keyValuePutHandler).Methods("PUT")
	r.HandleFunc("/v1/{key}", keyValueDeleteHandler).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":8080", r))
}
