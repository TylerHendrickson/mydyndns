// Package internal provides various utilities leveraged by the mydyndns CLI application.
package internal

import (
	"fmt"
	"sync"
)

// A StringCollection is a container for unique strings.
// It provides set-like operations for adding and removing members, as well for performing membership checks.
// All operations are atomic and thread-safe, making StringCollection appropriate for use in concurrent applications.
type StringCollection struct {
	m   map[string]struct{}
	mux sync.Mutex
}

// NewStringCollection returns a pointer to new StringCollection that is initialized with the provided
// variadic argument values as its members.
func NewStringCollection(members ...string) *StringCollection {
	sc := StringCollection{m: make(map[string]struct{}, len(members))}
	for _, it := range members {
		sc.Add(it)
	}
	return &sc
}

// Add takes a variadic argument of strings and adds them as new members of the StringCollection.
// If any provided members are already present in the StringCollection, they will be skipped,
// so it is not necessary to check for membership before calling Add.
func (sc *StringCollection) Add(members ...string) {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	for _, mem := range members {
		sc.m[mem] = struct{}{}
	}
}

// Remove takes a variadic argument of strings and removes them as members from the StringCollection.
// If any provided members are not currently present in the StringCollection, they will have
// no effect on the operation, so it is not necessary to check for membership before calling Remove.
func (sc *StringCollection) Remove(members ...string) {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	for _, mem := range members {
		delete(sc.m, mem)
	}
}

// Contains checks whether s is currently a member of the StringCollection.
func (sc *StringCollection) Contains(s string) bool {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	_, exists := sc.m[s]
	return exists
}

// Slice returns a snapshot of the StringCollection's member values as a new string slice.
func (sc *StringCollection) Slice() []string {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	s := make([]string, len(sc.m))
	i := 0
	for mem := range sc.m {
		s[i] = mem
		i++
	}
	return s
}

// String returns a string representing the member values of the StringCollection.
func (sc *StringCollection) String() string {
	return fmt.Sprint(sc.Slice())
}

// Len returns the size (member count) of the StringCollection.
func (sc *StringCollection) Len() int {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	return len(sc.m)
}
