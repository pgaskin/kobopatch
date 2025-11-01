package main

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestStringSlice(t *testing.T) {
	for _, c := range []struct {
		In  string
		Out []string
	}{
		{`asd`, []string{"asd"}},
		{`asd sdf`, []string{"asd sdf"}},
		{`[asd, sdf]`, []string{"asd", "sdf"}},
		{`["asd", "sdf"]`, []string{"asd", "sdf"}},
		{`
    - asd
    - sdf`, []string{"asd", "sdf"}},
		{`
    - "asd"
    - "sdf"`, []string{"asd", "sdf"}},
	} {
		t.Run(c.In, func(t *testing.T) {
			t.Parallel()
			var obj struct {
				Test stringSlice `yaml:"Test"`
			}
			if err := yaml.Unmarshal([]byte(fmt.Sprintf("Test: %s", c.In)), &obj); err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			t.Logf("out: %#v", obj)
			for _, ev := range c.Out {
				var found bool
				for _, av := range obj.Test {
					found = found || av == ev
				}
				if !found {
					t.Errorf("expected to find '%s' in output", ev)
				}
			}
		})
	}
}
