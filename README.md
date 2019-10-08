# persister

[![Go Report Card](https://goreportcard.com/badge/github.com/trandoshan-io/persister)](https://goreportcard.com/report/github.com/trandoshan-io/persister)

Persister is a Go written program designed to persist/archivate crawled resources

## features

- use scalable messaging protocol (nats)
- use scalable database system

## how it work

- The Persister process connect to a nats server (specified by env variable *NATS_URI*)
and set-up a subscriber for message with tag *contentSubject*
- When resource data is received the persister will aggregate the data if possible (f.e extract page title, etc...)
- Then data will be persisted to the database