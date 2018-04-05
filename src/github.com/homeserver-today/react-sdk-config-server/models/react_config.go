package models

import "github.com/imdario/mergo"

type ReactConfig map[string]interface{}

func (c *ReactConfig) ApplyDefaults(defaults *ReactConfig) (error) {
	return mergo.Merge(c, defaults)
}

func (c *ReactConfig) TakeFrom(source map[string]interface{}) (error) {
	return mergo.Map(c, source)
}
