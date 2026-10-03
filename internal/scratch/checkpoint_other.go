//go:build !unix

package scratch

import (
	"context"
	"errors"
)

// errUnsupportedSpool 은 spool 이 없는 OS 의 답이다 — runc-overlay 노드가 linux 에만 있다.
var errUnsupportedSpool = errors.New("the checkpoint spool is not supported on this platform")

// Open 은 이 OS 에서 지원하지 않는다.
func (s *Store) Open() (SpoolCheck, error) { return SpoolCheck{}, errUnsupportedSpool }

// SameFilesystem 은 이 OS 에서 지원하지 않는다.
func (s *Store) SameFilesystem() (bool, error) { return false, errUnsupportedSpool }

// Filesystem 은 이 OS 에서 지원하지 않는다.
func (s *Store) Filesystem() (Filesystem, error) { return Filesystem{}, errUnsupportedSpool }

// Kept 는 0 이다.
func (s *Store) Kept() int64 { return 0 }

// Reserve 는 이 OS 에서 지원하지 않는다.
func (s *Store) Reserve(Entry) (*Reservation, error) { return nil, errUnsupportedSpool }

// Commit 은 이 OS 에서 지원하지 않는다.
func (s *Store) Commit(*Reservation, Entry) (Entry, error) { return Entry{}, errUnsupportedSpool }

// Abandon 은 이 OS 에서 지원하지 않는다.
func (s *Store) Abandon(*Reservation) error { return errUnsupportedSpool }

// Dispose 는 이 OS 에서 지원하지 않는다.
func (s *Store) Dispose(*Reservation) error { return errUnsupportedSpool }

// Report 는 이 OS 에서 지원하지 않는다.
func (s *Store) Report(string, string) error { return errUnsupportedSpool }

// Settle 은 이 OS 에서 지원하지 않는다.
func (s *Store) Settle(context.Context) (SettleResult, error) {
	return SettleResult{}, errUnsupportedSpool
}

// Reconcile 은 이 OS 에서 지원하지 않는다.
func (s *Store) Reconcile() (ReconcileResult, error) { return ReconcileResult{}, errUnsupportedSpool }

// List 는 이 OS 에서 지원하지 않는다.
func (s *Store) List() ([]Listed, error) { return nil, errUnsupportedSpool }

// Lookup 은 이 OS 에서 지원하지 않는다.
func (s *Store) Lookup(string) (*Listed, error) { return nil, errUnsupportedSpool }
