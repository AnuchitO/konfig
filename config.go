package konfig

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lalamove/nui/nlogger"
	"github.com/prometheus/client_golang/prometheus"
)

var _ Store = (*store)(nil)

var (
	// ErrInvalidConfigFileFormat is the error returned when a problem is encountered when parsing the
	// config file
	ErrInvalidConfigFileFormat = errors.New("Err invalid file format")

	// ErrLoaderNotFound is the error thrown when the loader with the given name cannot be found in the config store
	ErrLoaderNotFound = errors.New("Err loader not found")
	// ErrConfigNotFoundMsg is the error message thrown when a config key is not set
	ErrConfigNotFoundMsg = "Err config '%s' not found"
	// ErrStrictKeyNotFoundMsg is the error returned when a strict key is not found in the konfig store
	ErrStrictKeyNotFoundMsg = "Err strict key '%s' not found"
)

const (
	missingConfMsg = "Config %s missing"
	defaultName    = "root"
)

// ErrMissingConfig is the type representing an error when a required config is missing
type ErrMissingConfig string

// Error implements the error interface
func (e ErrMissingConfig) Error() string {
	return string(e)
}

// DefaultConfig returns a default Config
func DefaultConfig() *Config {
	return &Config{
		ExitCode: 1,
		Logger:   nlogger.NewProvider(nlogger.New(os.Stdout, "CONFIG | ")),
		Name:     defaultName,
	}
}

// Config is the config to init a config store
type Config struct {
	// Name is the name of the config store, it is used in metrics as a label. When creating a config group, the name of the group becomes the name of the store
	Name string
	// ExitCode is the code to exit when errors are encountered in loaders
	ExitCode int
	// Disables exiting the program (os.Exit) when errors on loaders
	NoExitOnError bool
	// NoStopOnFailure if false the store closes all registered Watchers and Closers and exit the process unless NoExitOnError is true
	// when a Loader fails to load or a Loader Hook fails. If true, nothing happens when a Loader fails.
	NoStopOnFailure bool
	// Logger is the logger used internally
	Logger nlogger.Provider
	// Metrics sets whether a konfig.Store should record metrics for config loaders
	Metrics bool
}

// Store is the interface
type Store interface {
	// Name returns the name of the store
	Name() string
	// SetLogger sets the logger within the store
	// it will propagate to all children groups
	SetLogger(l nlogger.Structured)
	// RegisterLoader registers a Loader in the store and adds the given loader hooks.
	RegisterLoader(l Loader, loaderHooks ...func(Store) error) *ConfigLoader
	// RegisterLoaderWatcher reigsters a LoaderWatcher in the store and adds the given loader hooks.
	RegisterLoaderWatcher(lw LoaderWatcher, loaderHooks ...func(Store) error) *ConfigLoader
	// RegisterCloser registers an io.Closer in the store. A closer closes when konfig fails to load configs.
	RegisterCloser(closer io.Closer) Store
	// Strict specifies mandatory keys on the konfig. When Strict is called, konfig will check that the specified keys are present, else it will return a non nil error.
	// Then, after every following `Load` of a loader, it will check if the strict keys are still present in the konfig and consider the load a failure if a key is not present anymore.
	Strict(...string) Store
	// RunHooks runs all hooks and child groups hooks
	RunHooks() error

	// Load loads all loaders registered in the store. If it faisl it returns a non nil error
	Load() error
	// Watch starts all watchers registered in the store. If it fails it returns a non nil error.
	Watch() error

	// LoadWatch loads all loaders registered in the store, then starts watching all
	// watchers. If loading or starting watchers fails, loadwatch stops and returns a non nil error.
	LoadWatch() error

	// Group lazyloads a child Store from the current store. If the group already exists, it just returns it, else it creates it and returns it. Groups are useful to namespace configs by domain.
	Group(g string) Store

	// Get gets the value with the key k fron the store. If the key is not set, Get returns nil. To check wether a value is really set, use Exists.
	Get(k string) interface{}
	// MustGet tries to get the value with the key k from the store. If the key k does not exist in the store, MustGet panics.
	MustGet(k string) interface{}
	// Set sets the key k with the value v in the store.
	Set(k string, v interface{})
	// Exists checks wether the key k is set in the store.
	Exists(k string) bool
	// MustString tries to get the value with the key k from the store and casts it to a string. If the key k does not exist in the store, MustGet panics.
	MustString(k string) string

	// String tries to get the value with the key k from the store and casts it to a string. If the key k does not exist it returns the Zero value.
	String(k string) string

	// MustInt tries to get the value with the key k from the store and casts it to a int. If the key k does not exist in the store, MustInt panics.
	MustInt(k string) int

	// Int tries to get the value with the key k from the store and casts it to a int. If the key k does not exist it returns the Zero value.
	Int(k string) int

	// MustFloat tries to get the value with the key k from the store and casts it to a float. If the key k does not exist in the store, MustFloat panics.
	MustFloat(k string) float64
	// Float tries to get the value with the key k from the store and casts it to a float. If the key k does not exist it returns the Zero value.
	Float(k string) float64

	// MustBool tries to get the value with the key k from the store and casts it to a bool. If the key k does not exist in the store, MustBool panics.
	MustBool(k string) bool
	// Bool tries to get the value with the key k from the store and casts it to a bool. If the key k does not exist it returns the Zero value.
	Bool(k string) bool

	// MustDuration tries to get the value with the key k from the store and casts it to a time.Duration. If the key k does not exist in the store, MustDuration panics.
	MustDuration(k string) time.Duration
	// Duration tries to get the value with the key k from the store and casts it to a time.Duration. If the key k does not exist it returns the Zero value.
	Duration(k string) time.Duration

	// MustTime tries to get the value with the key k from the store and casts it to a time.Time. If the key k does not exist in the store, MustTime panics.
	MustTime(k string) time.Time
	// Time tries to get the value with the key k from the store and casts it to a time.Time. If the key k does not exist it returns the Zero value.
	Time(k string) time.Time

	// MustStringSlice tries to get the value with the key k from the store and casts it to a []string. If the key k does not exist in the store, MustStringSlice panics.
	MustStringSlice(k string) []string
	// StringSlice tries to get the value with the key k from the store and casts it to a []string. If the key k does not exist it returns the Zero value.
	StringSlice(k string) []string

	// MustIntSlice tries to get the value with the key k from the store and casts it to a []int. If the key k does not exist in the store, MustIntSlice panics.
	MustIntSlice(k string) []int
	// IntSlice tries to get the value with the key k from the store and casts it to a []int. If the key k does not exist it returns the Zero value.
	IntSlice(k string) []int

	// MustStringMap tries to get the value with the key k from the store and casts it to a map[string]interface{}. If the key k does not exist in the store, MustStringMap panics.
	MustStringMap(k string) map[string]interface{}
	// StringMap tries to get the value with the key k from the store and casts it to a map[string]interface{}. If the key k does not exist it returns the Zero value.
	StringMap(k string) map[string]interface{}

	// MustStringMapString tries to get the value with the key k from the store and casts it to a map[string]string. If the key k does not exist in the store, MustStringMapString panics.
	MustStringMapString(k string) map[string]string
	// StringMapString tries to get the value with the key k from the store and casts it to a map[string]string. If the key k does not exist it returns the Zero value.
	StringMapString(k string) map[string]string

	// Bind binds a value (either a map[string]interface{} or a struct) to the config store. When config values are set on the config store, they are also set on the bound value.
	Bind(interface{})

	// Value returns the value bound to the config store.
	// It panics if no bound value has been set
	Value() interface{}
}

// store is the concrete implementation of the Store
type store struct {
	name       string
	cfg        *Config
	m          *atomic.Value
	mut        *sync.Mutex
	groups     map[string]*store
	v          *value
	metrics    map[string]prometheus.Collector
	strictKeys []string
	loaded     bool

	WatcherLoaders []*loaderWatcher
	WatcherClosers Closers
	Closers        Closers
}

var (
	c    *store
	once sync.Once
)

// Init initiates the global config store with the given Config cfg
func Init(cfg *Config) {
	c = newStore(cfg)
}

// New returns a new Store with the given config
func New(cfg *Config) Store {
	return newStore(cfg)
}

// SetLogger sets the logger used in the global store
func SetLogger(l nlogger.Structured) {
	instance().SetLogger(l)
}
func (c *store) SetLogger(l nlogger.Structured) {
	c.cfg.Logger.Replace(l)
}

func (c *store) Name() string {
	return c.name
}

// RegisterLoader registers a Loader l with a given Watcher w.
func RegisterLoader(l Loader, loaderHooks ...func(Store) error) *ConfigLoader {
	return instance().RegisterLoader(l, loaderHooks...)
}
func (c *store) RegisterLoader(l Loader, loaderHooks ...func(Store) error) *ConfigLoader {
	var lw = c.newLoaderWatcher(l, NopWatcher{}, loaderHooks)

	c.WatcherLoaders = append(
		c.WatcherLoaders,
		lw,
	)

	return c.newConfigLoader(lw)
}

// RegisterLoaderWatcher registers a WatcherLoader to the config.
func RegisterLoaderWatcher(lw LoaderWatcher, loaderHooks ...func(Store) error) *ConfigLoader {
	return instance().RegisterLoaderWatcher(lw, loaderHooks...)
}
func (c *store) RegisterLoaderWatcher(lw LoaderWatcher, loaderHooks ...func(Store) error) *ConfigLoader {
	var lwatcher = c.newLoaderWatcher(lw, lw, loaderHooks)

	c.WatcherClosers = append(c.WatcherClosers, lw)
	c.WatcherLoaders = append(
		c.WatcherLoaders,
		lwatcher,
	)

	return c.newConfigLoader(lwatcher)
}

// RegisterCloser adds a closer to the list of closers.
// Closers are closed when an error occured while reloading a config and the ExitOnError config is set to true
func RegisterCloser(closer io.Closer) Store {
	return instance().RegisterCloser(closer)
}
func (c *store) RegisterCloser(closer io.Closer) Store {
	c.Closers = append(c.Closers, closer)
	return c
}

// Strict specifies mandatory keys on the konfig. After strict is called, konfig will wait for the first config Load to happen and will check if the
// specified strict keys are present, if not, Load will return a non nil error. Then, after every following `Load` of a loader, it will check if the strict keys are still present in the konfig and consider the load a failure if a key is not present anymore.
func Strict(keys ...string) Store {
	return instance().Strict(keys...)
}
func (c *store) Strict(keys ...string) Store {
	c.strictKeys = keys
	return c
}
func (c *store) checkStrictKeys() error {
	for _, k := range c.strictKeys {
		if !c.Exists(k) {
			return fmt.Errorf(ErrStrictKeyNotFoundMsg, k)
		}
	}
	return nil
}

// RunHooks runs all hooks and child groups hooks
func RunHooks() error {
	return instance().RunHooks()
}
func (c *store) RunHooks() error {
	// run all hooks
	for _, wl := range c.WatcherLoaders {
		if wl.loaderHooks != nil {
			if err := wl.loaderHooks.Run(c); err != nil {
				return err
			}
		}
	}

	// run hooks on chil groups
	if c.groups != nil {
		for _, gr := range c.groups {
			if err := gr.RunHooks(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Instance returns the singleton global config store
func Instance() Store {
	if c == nil {
		c = newStore(DefaultConfig())
	}
	return c
}

// Stop stops the config store
func (c *store) stop() {
	if err := c.WatcherClosers.Close(); err != nil {
		c.cfg.Logger.Get().Error(err.Error())
	}

	if err := c.Closers.Close(); err != nil {
		c.cfg.Logger.Get().Error(err.Error())
	}

	// exit on error unless specified
	if !c.cfg.NoExitOnError {
		os.Exit(c.cfg.ExitCode)
	}
}

func instance() *store {
	if c == nil {
		c = newStore(DefaultConfig())
	}
	return c
}

func reset() {
	var cc = instance()
	if cc != nil {
		c = newStore(c.cfg)
	}
}

func newStore(cfg *Config) *store {
	// check if logger exists, else set default logger
	if cfg.Logger == nil {
		cfg.Logger = defaultLogger()
	}

	var mValue atomic.Value
	var m = make(s)
	mValue.Store(m)

	var s = &store{
		name:           cfg.Name,
		m:              &mValue,
		cfg:            cfg,
		mut:            &sync.Mutex{},
		groups:         make(map[string]*store),
		WatcherLoaders: make([]*loaderWatcher, 0, 10),
		WatcherClosers: make(Closers, 0, 10),
		Closers:        make(Closers, 0, 10),
	}

	if s.name == "" {
		s.name = defaultName
	}

	// init metrics if it is enabled
	if cfg.Metrics {
		s.initMetrics()
	}

	return s
}

func defaultLogger() nlogger.Provider {
	return nlogger.NewProvider(nlogger.New(os.Stdout, "CONFIG | "))
}
