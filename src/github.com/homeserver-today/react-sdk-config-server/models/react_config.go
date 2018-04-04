package models

import "github.com/imdario/mergo"

type ReactConfig map[string]interface{}

func (c *ReactConfig) ApplyDefaults(defaults *ReactConfig) (error) {
	return mergo.Merge(c, defaults)
}
