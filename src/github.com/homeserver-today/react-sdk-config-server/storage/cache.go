package storage

import (
	"time"
	"github.com/patrickmn/go-cache"
	"context"
	"github.com/sirupsen/logrus"
	"sync"
	"github.com/homeserver-today/react-sdk-config-server/models"
	"github.com/ryanuber/go-glob"
	"database/sql"
	"strings"
	"sort"
)

const cacheExpiration = 1 * time.Hour
const cleanupInterval = 2 * time.Hour
const domainPrefix = "domain_"
const globListKey = "globs"

type configCacheFactory struct {
	cache *cache.Cache
}

type configCache struct {
	cache *cache.Cache
	ctx   context.Context
	log   *logrus.Entry
}

var cacheInstance *configCacheFactory
var cacheSingletonLock = &sync.Once{}

func getBaseCache() (*configCacheFactory) {
	if cacheInstance == nil {
		cacheSingletonLock.Do(func() {
			cacheInstance = &configCacheFactory{
				cache: cache.New(cacheExpiration, cleanupInterval),
			}
		})
	}

	return cacheInstance
}

func GetForwardingCache(ctx context.Context, log *logrus.Entry) (*configCache) {
	return &configCache{
		cache: getBaseCache().cache,
		ctx:   ctx,
		log:   log,
	}
}

func (c *configCache) GetConfig(domain string) (*models.ReactConfig, error) {
	config, found := c.cache.Get(domainPrefix + domain)
	if found {
		return config.(*models.ReactConfig), nil
	}

	if strings.Contains(domain, "*") {
		dbConfig, err := GetDatabase().GetConfig(c.ctx, domain)
		if err == sql.ErrNoRows {
			dbConfig = models.ReactConfig{}
		} else if err != nil {
			return nil, err
		}

		return &dbConfig, nil
	} else {
		calcConfig, err := c.calculateConfig(domain)
		if err != nil {
			return nil, err
		}

		c.cache.Set(domainPrefix+domain, calcConfig, cache.DefaultExpiration)
		return calcConfig, nil
	}
}

func (c *configCache) calculateConfig(domain string) (*models.ReactConfig, error) {
	c.log.Info("Calculating the complete config for domain " + domain)

	globs, found := c.cache.Get(globListKey)
	if !found {
		var err error
		globs, err = GetDatabase().ListGlobs(c.ctx)
		if err != nil {
			return nil, err
		}

		c.cache.Set(globListKey, globs, cache.DefaultExpiration)
	}

	type templateTuple struct {
		template *models.ReactConfig
		weight   int
	}
	templates := make([]templateTuple, 0)
	for _, template := range globs.([]string) {
		if !glob.Glob(template, domain) {
			continue
		}

		config, err := c.GetConfig(template)
		if err != nil {
			return nil, err
		}

		weight := 0
		if (*config)["hstoday.weight"] != nil {
			weight = int((*config)["hstoday.weight"].(float64))
		}
		templates = append(templates, templateTuple{config, weight})
	}

	// Sort the templates so we have the least important first
	sort.Slice(templates, func(i int, j int) bool {
		return templates[i].weight < templates[j].weight
	})

	// Build the default config
	domainDefaults := &models.ReactConfig{}
	for _, t := range templates {
		t.template.ApplyDefaults(domainDefaults)
		domainDefaults = t.template
	}

	// Now that we have a default config for the domain, we can apply it to the domain's real config
	dbConfig, err := GetDatabase().GetConfig(c.ctx, domain)
	if err == sql.ErrNoRows {
		dbConfig = models.ReactConfig{}
	} else if err != nil {
		return nil, err
	}

	dbConfig.ApplyDefaults(domainDefaults)
	return &dbConfig, nil
}

func (c *configCache) SetConfig(domain string, config *models.ReactConfig) (*models.ReactConfig, error) {
	var err error
	if config != nil {
		c.log.Info("Updating config for " + domain)
		err = GetDatabase().UpsertConfig(c.ctx, domain, *config)
	} else {
		c.log.Info("Deleting config for " + domain)
		err = GetDatabase().DeleteConfig(c.ctx, domain)
	}
	if err != nil {
		return nil, err
	}

	// Purge the domain from the cache
	c.cache.Delete(domainPrefix + domain)

	// If the domain is a glob, clear the cache entirely. It is relatively rare that people will be
	// setting glob configs, so this shouldn't be that bad. It also means we don't need to track which
	// globs were used where.
	if strings.Contains(domain, "*") {
		for k := range c.cache.Items() {
			c.cache.Delete(k)
		}
	}

	return c.GetConfig(domain)
}

func (c *configCache) DeleteConfig(domain string) (*models.ReactConfig, error) {
	return c.SetConfig(domain, nil)
}
