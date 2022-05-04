package main

import (
	"sort"
	"strings"
)

type stringSet map[string]struct{}

func newStringSet(elements ...string) stringSet {
	s := make(stringSet, len(elements))
	for _, element := range elements {
		s[element] = struct{}{}
	}
	return s
}

func (s stringSet) Set(value string) error {
	for element := range s {
		delete(s, element)
	}
	for _, element := range strings.Split(value, ",") {
		s[element] = struct{}{}
	}
	return nil
}

func (s stringSet) String() string {
	elements := make([]string, 0, len(s))
	for element := range s {
		elements = append(elements, element)
	}
	sort.Strings(elements)
	return strings.Join(elements, ",")
}

func (s stringSet) contains(element string) bool {
	_, ok := s[element]
	return ok
}

func (s stringSet) subtract(other stringSet) stringSet {
	result := make(stringSet)
	for element := range s {
		if _, ok := other[element]; !ok {
			result[element] = struct{}{}
		}
	}
	return result
}
