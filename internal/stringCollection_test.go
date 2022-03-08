package internal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStringCollection(t *testing.T) {
	for ti, tt := range [][]string{{}, {"a"}, {"a", "b", "c"}} {
		t.Run(fmt.Sprint(ti), func(t *testing.T) {
			var keys []string
			for k := range NewStringCollection(tt...).m {
				keys = append(keys, k)
			}
			assert.ElementsMatch(t, tt, keys)
		})
	}
}

func TestStringCollection_Add(t *testing.T) {
	for ti, tt := range []struct{ start, add []string }{
		{[]string{"a", "b", "c"}, []string{"z"}},
		{[]string{"a", "b", "c"}, []string{"z", "x"}},
		{[]string{"a", "b", "c"}, []string{}},
		{[]string{}, []string{"z", "x", "y"}},
	} {
		t.Run(fmt.Sprint(ti), func(t *testing.T) {
			sc := NewStringCollection(tt.start...)
			sc.Add(tt.add...)
			for _, member := range tt.add {
				assert.Contains(t, sc.m, member, "missing member key %q in underlying map", member)
			}
		})
	}
}

func TestStringCollection_Remove(t *testing.T) {
	for ti, tt := range []struct{ start, remove []string }{
		{[]string{"a", "b", "c"}, []string{"z"}},
		{[]string{"a", "b", "c"}, []string{"a"}},
		{[]string{"a", "b", "c"}, []string{"a", "b"}},
		{[]string{"a", "b", "c"}, []string{}},
		{[]string{}, []string{"z", "x", "y"}},
	} {
		t.Run(fmt.Sprint(ti), func(t *testing.T) {
			sc := NewStringCollection(tt.start...)
			sc.Remove(tt.remove...)
			for _, notMember := range tt.remove {
				assert.NotContainsf(t, sc.m, notMember, "unexpected member key %q in underlying map", notMember)
			}
		})
	}
}

func TestStringCollection_Contains(t *testing.T) {
	for ti, tt := range []struct{ start, add, remove, expectContains, expectNotContains []string }{
		{
			[]string{"a"},
			[]string{"b", "c"},
			[]string{"b"},
			[]string{"a", "c"},
			[]string{"b"},
		},
		{
			[]string{"a"},
			[]string{"b"},
			[]string{"a"},
			[]string{"b"},
			[]string{"a"},
		},
	} {
		t.Run(fmt.Sprint(ti), func(t *testing.T) {
			sc := NewStringCollection(tt.start...)
			sc.Add(tt.add...)
			sc.Remove(tt.remove...)
			for _, member := range tt.expectContains {
				assert.True(t, sc.Contains(member),
					"StringCollection unexpectedly does not contain member %q", member)
			}
			for _, notMember := range tt.expectNotContains {
				assert.False(t, sc.Contains(notMember),
					"StringCollection unexpectedly contains member %q", notMember)
			}
		})
	}
}

func TestStringCollection_Slice(t *testing.T) {
	for ti, tt := range []struct{ start, expected []string }{
		{[]string{"a", "b"}, []string{"a", "b"}},
		{[]string{"up", "up", "down", "down", "a", "b", "b", "a"}, []string{"up", "down", "a", "b"}},
	} {
		t.Run(fmt.Sprint(ti), func(t *testing.T) {
			assert.ElementsMatch(t, tt.expected, NewStringCollection(tt.start...).Slice())
		})
	}
}

func TestStringCollection_String(t *testing.T) {
	for ti, tt := range [][]string{{"a"}, {"a", "b"}, {"a", "b", "c"}, {}} {
		t.Run(fmt.Sprint(ti), func(t *testing.T) {
			s := NewStringCollection(tt...).String()
			if len(tt) == 0 {
				assert.Equal(t, s, fmt.Sprintf("%s", []string{}),
					"String not represented as an empty slice")
			} else {
				assert.Equal(t, "[", s[0:1], "Does not start with opening brace")
				assert.Equal(t, "]", s[len(s)-1:], "Does not end with closing brace")
				assert.ElementsMatch(t, strings.Split(s[1:len(s)-1], " "), tt,
					"Slice representation does not match all elements")
			}
		})
	}
}

func TestStringCollection_Len(t *testing.T) {
	for _, tt := range []struct {
		name     string
		members  []string
		expected int
	}{
		{"Unique", []string{"a", "b", "c"}, 3},
		{"Empty", []string{}, 0},
		{"Duplicates", []string{"up", "up", "down", "down", "a", "b", "b", "a"}, 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, NewStringCollection(tt.members...).Len())
		})
	}
}
