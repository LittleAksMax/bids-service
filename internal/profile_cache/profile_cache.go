package profile_cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/LittleAksMax/bids-service/internal/services"
	"github.com/allegro/bigcache/v3"
	"github.com/google/uuid"
)

const (
	profileCacheLifeWindow   = 15 * time.Minute
	profileCacheCleanWindow  = time.Minute
	profileCacheMaxEntrySize = 2048
)

var errProfileCacheNotConfigured = errors.New("profile cache is not configured")

// ProfileCache is the facade of the cache
type ProfileCache interface {
	GetProfile(ctx context.Context, userID uuid.UUID, profileID int64) (*services.RegionProfile, error)
	Close() error
}

// profileFetchFunc is the type of function which gets all user profiles
type profileFetchFunc func(context.Context, uuid.UUID) ([]services.RegionProfile, error)

// profileLoadState tracks one in-flight refill for a single user.
type profileLoadState struct {
	done chan struct{}
	err  error
}

// profileLoadCoordinator stops a cache-miss stampede.
// If several workers miss the cache for the same user at the same time, we want exactly one of
// them to fetch all profiles while the others wait for that work to finish.
type profileLoadCoordinator struct {
	mu    sync.Mutex
	loads map[uuid.UUID]*profileLoadState
}

// bigcacheProfileCache is the concrete implementation behind ProfileCache
type bigcacheProfileCache struct {
	cache         *bigcache.BigCache
	loads         profileLoadCoordinator
	fetchProfiles profileFetchFunc
}

func NewProfileCache(ctx context.Context, userService *services.UserServiceClient) (ProfileCache, error) {
	cacheConfig := bigcache.DefaultConfig(profileCacheLifeWindow)
	cacheConfig.CleanWindow = profileCacheCleanWindow
	cacheConfig.MaxEntrySize = profileCacheMaxEntrySize
	cacheConfig.Verbose = false

	cache, err := bigcache.New(ctx, cacheConfig)
	if err != nil {
		return nil, fmt.Errorf("create profile cache: %w", err)
	}

	return &bigcacheProfileCache{
		cache: cache,
		fetchProfiles: func(ctx context.Context, userID uuid.UUID) ([]services.RegionProfile, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			// A miss for one profile means we make the service request to cache all profiles
			log.Printf("profile cache miss for user %s; fetching all profiles", userID)
			return userService.GetProfiles(ctx, userID)
		},
	}, nil
}

func (c *bigcacheProfileCache) Close() error {
	return c.cache.Close()
}

// Load runs the loader once per user at a time.
// The first goroutine becomes the loader and later goroutines for the same user just wait for the
// shared result instead of refilling the cache again.
func (c *profileLoadCoordinator) Load(ctx context.Context, userID uuid.UUID, loader func(context.Context) error) error {
	c.mu.Lock()
	if c.loads == nil {
		// The zero value should still be usable, so build the map lazily
		c.loads = make(map[uuid.UUID]*profileLoadState)
	}

	loadState, waitingForExistingLoad := c.loads[userID]
	if !waitingForExistingLoad {
		// No refill is running for this user yet, so this goroutine becomes the owner of the load.
		loadState = &profileLoadState{done: make(chan struct{})}
		c.loads[userID] = loadState
	}
	c.mu.Unlock()

	// some other goroutine is already fetching user profiles, so we wait for its results
	// (or cancellation)
	if waitingForExistingLoad {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-loadState.done:
			return loadState.err
		}
	}

	// Fetch all profiles using the callback
	err := loader(ctx)

	c.mu.Lock()
	loadState.err = err     // Set error (if exists)
	delete(c.loads, userID) // Delete entry
	close(loadState.done)   // Close channel
	c.mu.Unlock()

	return err
}

// GetProfile is the main lookup path used by processors.
// It first tries the cache directly. On a miss, it triggers a user-level refill and then reads
// the requested profile back out of the cache.
func (c *bigcacheProfileCache) GetProfile(ctx context.Context, userID uuid.UUID, profileID int64) (*services.RegionProfile, error) {
	if c == nil {
		return nil, errProfileCacheNotConfigured
	}

	// Fast path: the specific profile is already cached.
	profile, err := c.getCachedProfile(userID, profileID)
	switch {
	case err == nil:
		return profile, nil
	case !errors.Is(err, bigcache.ErrEntryNotFound):
		// A real (non-NotFound) cache error should fail immediately
		return nil, err
	}

	// The profile is missing, so refill that user's cached profiles once.
	loaderWithUserID := func(ctx context.Context) error {
		return c.loader(ctx, userID)
	}
	if err := c.loads.Load(ctx, userID, loaderWithUserID); err != nil {
		return nil, err
	}

	// Refilled cache, should be able to fetch profiles from it
	profile, err = c.getCachedProfile(userID, profileID)
	if errors.Is(err, bigcache.ErrEntryNotFound) {
		return nil, fmt.Errorf("profile %d not found for user %s", profileID, userID)
	}

	return profile, err
}

// loader does the actual refill work for one user.
// It fetches the full profile list from Amazon Ads API and then stores results in cache.
func (c *bigcacheProfileCache) loader(loadCtx context.Context, userID uuid.UUID) error {
	profiles, err := c.fetchProfiles(loadCtx, userID)
	if err != nil {
		return err
	}

	// Cache each fetched profile
	for _, profile := range profiles {
		// bigcache stores bytes, so we serialize each profile before writing it.
		entry, err := json.Marshal(profile)
		if err != nil {
			return fmt.Errorf("encode cached profile %d for user %s: %w", profile.ProfileID, userID, err)
		}

		// Each profile gets its own key, even though the refill fetched the whole user.
		if err := c.cache.Set(profileCacheKey(userID, profile.ProfileID), entry); err != nil {
			return fmt.Errorf("cache profile %d for user %s: %w", profile.ProfileID, userID, err)
		}
	}

	return nil
}

func (c *bigcacheProfileCache) getCachedProfile(userID uuid.UUID, profileID int64) (*services.RegionProfile, error) {
	entry, err := c.cache.Get(profileCacheKey(userID, profileID))
	if err != nil {
		return nil, err
	}

	var profile services.RegionProfile
	if err := json.Unmarshal(entry, &profile); err != nil {
		return nil, fmt.Errorf("decode cached profile %d for user %s: %w", profileID, userID, err)
	}

	return &profile, nil
}

// profileCacheKey combines the user ID and profile ID into one stable cache key
func profileCacheKey(userID uuid.UUID, profileID int64) string {
	return userID.String() + ":" + strconv.FormatInt(profileID, 10)
}
