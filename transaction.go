package main

import (
	"bufio"
	"fmt"
	"os"
)

type EventType byte

const (
	_                     = iota // iota == 0; ignore this value
	EventDelete EventType = iota // iota == 1
	EventPut    EventType = iota // iota == 2; implicitly repeat last
)

type Event struct {
	Sequence  uint64
	EventType EventType
	Key       string
	Value     string
}

type TransactionLogger interface {
	WriteDelete(key string)
	WritePut(key, value string)
	Err() <-chan error
	ReadEvents() (<-chan Event, <-chan error)
	Run()
}

type FileTransactionLogger struct {
	events       chan<- Event // Write-only channel for sending events
	errors       <-chan error
	lastSequence uint64   // The last used event sequence number
	file         *os.File // The location of the transaction log
}

func NewFileTransactionLogger(filename string) (TransactionLogger, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		return nil, fmt.Errorf("cannot open transaction log file: %w", err)
	}

	return &FileTransactionLogger{file: file}, nil
}

func (l *FileTransactionLogger) WritePut(key, value string) {
	l.events <- Event{EventType: EventPut, Key: key, Value: value}
}

func (l *FileTransactionLogger) WriteDelete(key string) {
	l.events <- Event{EventType: EventDelete, Key: key}
}

func (l *FileTransactionLogger) Err() <-chan error {
	return l.errors
}

func (l *FileTransactionLogger) Run() {
	events := make(chan Event, 16) // Make an events channel
	l.events = events

	errors := make(chan error, 1) // Make an erros channel
	l.errors = errors

	go func() {
		for e := range events { // Retrieve the next Event

			l.lastSequence++ // Increment sequence number

			_, err := fmt.Fprintf(
				l.file,
				"%d\t%d\t%s\t%s\n",
				l.lastSequence, e.EventType, e.Key, e.Value)

			if err != nil {
				errors <- err
				return
			}
		}
	}()
}

func (l *FileTransactionLogger) ReadEvents() (<-chan Event, <-chan error) {
	scanner := bufio.NewScanner(l.file) // Create a scanner gor l.file
	outEvent := make(chan Event)        // An unbeffered Event channel
	outError := make(chan error, 1)     // A buffered error channel

	go func() {
		var e Event
		defer close(outEvent)
		defer close(outError)

		for scanner.Scan() {
			line := scanner.Text()

			if _, err := fmt.Sscanf(line, "%d\t%d\t%s\t%s", &e.Sequence, &e.EventType, &e.Key, &e.Value); err != nil {

				outError <- fmt.Errorf("input parse error: %w", err)
				return
			}

			if l.lastSequence >= e.Sequence {
				outError <- fmt.Errorf("transaction numbers out of sequence")
				return
			}

			l.lastSequence = e.Sequence

			outEvent <- e
		}

		if err := scanner.Err(); err != nil {
			outError <- fmt.Errorf("transaction log read failure: %w", err)
		}
	}()

	return outEvent, outError
}
