package main

import (
	"io/ioutil"
	"regexp"
	"strings"
)

type Config struct {
	m map[string]string
}

func ReadConfig(name string) Config {
	config := Config{
		m: make(map[string]string),
	}

	expr := regexp.MustCompile(`^\s*(\S+)\s+(\S+)\s*$`)

	data, err := ioutil.ReadFile("/etc/linode/longview.d/" + name + ".conf")
	if err == nil {
		for _, l := range strings.Split(string(data), "\n") {
			i := strings.IndexByte(l, '#')
			if i >= 0 {
				l = l[:i]
			}

			if m := expr.FindStringSubmatch(l); len(m) == 3 {
				key := m[1]
				value := m[2]

				config.m[key] = value
			}
		}
	}

	return config
}

func (c *Config) GetOrDefault(key string, def string) string {
	value, ok := c.m[key]
	if ok {
		return value
	}
	return def
}
