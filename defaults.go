package pocket

import (
	"context"
	"sync"
	"time"
)

// globalDefaults holds the default configuration for all nodes.
var globalDefaults = &nodeDefaults{
	prep:       defaultPrep,
	exec:       defaultExec,
	post:       defaultPost,
	retryDelay: 100 * time.Millisecond,
}

// nodeDefaults contains default configuration that can be applied to nodes.
type nodeDefaults struct {
	mu sync.RWMutex
	
	// Lifecycle defaults
	prep PrepFunc
	exec ExecFunc
	post PostFunc
	
	// Options defaults
	maxRetries  int
	retryDelay  time.Duration
	timeout     time.Duration
	onError     func(error)
	fallback    func(ctx context.Context, input any, err error) (any, error)
	onSuccess   func(ctx context.Context, store StoreWriter, output any)
	onFailure   func(ctx context.Context, store StoreWriter, err error)
	onComplete  func(ctx context.Context, store StoreWriter)
}

// SetDefaults configures global defaults for all nodes.
func SetDefaults(opts ...Option) {
	globalDefaults.mu.Lock()
	defer globalDefaults.mu.Unlock()
	
	// Create a temporary nodeOptions to apply options to
	tempOpts := nodeOptions{
		prep:       globalDefaults.prep,
		exec:       globalDefaults.exec,
		post:       globalDefaults.post,
		maxRetries: globalDefaults.maxRetries,
		retryDelay: globalDefaults.retryDelay,
		timeout:    globalDefaults.timeout,
		onError:    globalDefaults.onError,
		fallback:   globalDefaults.fallback,
		onSuccess:  globalDefaults.onSuccess,
		onFailure:  globalDefaults.onFailure,
		onComplete: globalDefaults.onComplete,
	}
	
	// Apply options
	for _, opt := range opts {
		opt(&tempOpts)
	}
	
	// Copy back to globalDefaults
	if tempOpts.prep != nil {
		globalDefaults.prep = tempOpts.prep
	}
	if tempOpts.exec != nil {
		globalDefaults.exec = tempOpts.exec
	}
	if tempOpts.post != nil {
		globalDefaults.post = tempOpts.post
	}
	globalDefaults.maxRetries = tempOpts.maxRetries
	globalDefaults.retryDelay = tempOpts.retryDelay
	globalDefaults.timeout = tempOpts.timeout
	globalDefaults.onError = tempOpts.onError
	globalDefaults.fallback = tempOpts.fallback
	globalDefaults.onSuccess = tempOpts.onSuccess
	globalDefaults.onFailure = tempOpts.onFailure
	globalDefaults.onComplete = tempOpts.onComplete
}

// SetDefaultPrep sets the global default prep function.
func SetDefaultPrep(fn PrepFunc) {
	globalDefaults.mu.Lock()
	defer globalDefaults.mu.Unlock()
	globalDefaults.prep = fn
}

// SetDefaultExec sets the global default exec function.
func SetDefaultExec(fn ExecFunc) {
	globalDefaults.mu.Lock()
	defer globalDefaults.mu.Unlock()
	globalDefaults.exec = fn
}

// SetDefaultPost sets the global default post function.
func SetDefaultPost(fn PostFunc) {
	globalDefaults.mu.Lock()
	defer globalDefaults.mu.Unlock()
	globalDefaults.post = fn
}

// getDefaults returns a copy of the current global defaults.
func getDefaults() (prep PrepFunc, exec ExecFunc, post PostFunc, opts nodeOptions) {
	globalDefaults.mu.RLock()
	defer globalDefaults.mu.RUnlock()
	
	return globalDefaults.prep, 
		globalDefaults.exec, 
		globalDefaults.post,
		nodeOptions{
			maxRetries:  globalDefaults.maxRetries,
			retryDelay:  globalDefaults.retryDelay,
			timeout:     globalDefaults.timeout,
			onError:     globalDefaults.onError,
			fallback:    globalDefaults.fallback,
			onSuccess:   globalDefaults.onSuccess,
			onFailure:   globalDefaults.onFailure,
			onComplete:  globalDefaults.onComplete,
		}
}

// ResetDefaults resets all global defaults to their initial values.
func ResetDefaults() {
	globalDefaults.mu.Lock()
	defer globalDefaults.mu.Unlock()
	
	globalDefaults.prep = defaultPrep
	globalDefaults.exec = defaultExec
	globalDefaults.post = defaultPost
	globalDefaults.maxRetries = 0
	globalDefaults.retryDelay = 100 * time.Millisecond
	globalDefaults.timeout = 0
	globalDefaults.onError = nil
	globalDefaults.fallback = nil
	globalDefaults.onSuccess = nil
	globalDefaults.onFailure = nil
	globalDefaults.onComplete = nil
}