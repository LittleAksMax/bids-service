# Bids Service

This service looks for due bid schedules, gathers the data needed to evaluate them, runs the policy scripts, optionally updates Amazon Ads live, and records the bid changes.

## Packages

- `main`: starts everything up, builds the clients, creates the workers, and handles shutdown.
- `internal/config`: reads config from environment variables.
- `internal/services`: holds the HTTP clients and request/response types for the user service and policy service.
- `internal/profile_cache`: caches profile lookups and makes sure only one fetch per user is in progress at a time.
- `internal/receiver`: polls for due schedules, gets the profile and token context, marks work as processing, and sends jobs to processors.
- `internal/processors`: does the actual work: fetches reports and policy data, evaluates scripts, publishes bid changes, and updates schedule state.

## Caching

Profile caching lives in `internal/profile_cache` and has two parts:

1. `bigcache` stores cached profiles by a composite key:

```go
func profileCacheKey(userID uuid.UUID, profileID int64) string {
	return userID.String() + ":" + strconv.FormatInt(profileID, 10)
}
```

2. A per-user load coordinator makes sure repeated misses do not trigger repeated fetches.

The main idea is simple: if one profile is missing, the service fetches all profiles for that user, then stores them one by one in the cache. That way the next lookup for the same user is usually already there.

The concurrency control is handled by `profileLoadCoordinator`. It keeps a `loads` map keyed by `userID`, where each active fetch has:

- a `done` channel
- an `err` field
- ownership of the current fetch

The first goroutine that misses the cache for a user becomes the one that does the fetch. If other goroutines miss the cache for the same user while that is happening, they do not make another request. They just wait on the same `done` channel:

```go
loadState, waiting := c.loads[userID]
if !waiting {
	loadState = &profileLoadState{done: make(chan struct{})}
	c.loads[userID] = loadState
}
```

Those waiting goroutines block here:

```go
select {
case <-ctx.Done():
	return ctx.Err()
case <-loadState.done:
	return loadState.err
}
```

When the fetch finishes, it stores the shared result, removes the in-flight entry, and closes the channel:

```go
loadState.err = err
delete(c.loads, userID)
close(loadState.done)
```

In practice, this gives you:

- fast reads when the profile is already cached
- only one profile fetch per user at a time
- no wasted duplicate requests while another goroutine is already fetching
- waiting that still respects context cancellation

## Receiver and Processor Interaction

The receiver and processors split the work pretty cleanly:

- The receiver finds due schedules and turns them into processor jobs.
- A processor takes one of those jobs and does the real work.

At a high level:

1. The receiver gets due schedules from the user service.
2. For each schedule, it gets the profile from the profile cache, fetches the user tokens, picks the right regional refresh token, marks the schedule as `PROCESSING`, and sends the job to one processor.
3. The chosen processor sets its own Ads client region and refresh token from the message.
4. The processor then runs two things in parallel:
   - report generation and polling from Amazon
   - preparation of attached-policy, policy, and ad-group data
5. When both sides finish, the processor decodes the report rows, evaluates the policies, publishes any bid changes, and then drives the schedule to its next state.

That parallel split matters because:

- the report side is mostly waiting for Amazon
- the preparation side is mostly fetching internal data and building lookups

Doing both at the same time keeps the total processing time down. The overall model stays simple: the receiver schedules work, and the processor owns the rest.
