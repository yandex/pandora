// package core defines pandora engine extension points.
// Core interfaces implementations MAY be used for custom engine creation and using as a library,
// or MAY be registered in pandora plugin system (look at core/plugin package), for creating engine
// from abstract config.
//
// The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT",
// "RECOMMENDED",  "MAY", and "OPTIONAL" in that package doc are to be interpreted as described in
// https://www.ietf.org/rfc/rfc2119.txt
package core

import (
	"context"
	"io"
	"time"

	"go.uber.org/zap"
)

// Ammo is data required for one shot. SHOULD contains something that differs
// from one shot to another. Something like requested recourse indetificator, query params,
// meta information helpful for future shooting analysis.
// Information common for all shoots SHOULD be passed via Provider configuration.
type Ammo interface{}

// ResettableAmmo is ammo that can be efficiently reset before reuse.
// Generic Provider (Provider that accepts undefined type of Ammo) SHOULD Reset Ammo before reuse
// if it implements ResettableAmmo.
// Ammo that is not going to be used with generic Providers don't need to implement ResettableAmmo.
type ResettableAmmo interface {
	Ammo
	Reset()
}

//go:generate mockery --name=Provider --case=underscore --outpkg=coremock

// Provider is routine that generates ammo for Instance shoots.
// A Provider MUST be goroutine safe.
type Provider interface {
	// Run starts provider routine of ammo  generation.
	// Blocks until ammo finish, error or context cancel.
	// Run MUST be called only once. Run SHOULD be called before Acquire or Release calls, but
	// MAY NOT because of goroutine races.
	// In case of ctx cancel, SHOULD return nil, but MAY ctx.Err(), or error caused ctx.Err()
	// in terms of github.com/pkg/errors.Cause.
	Run(ctx context.Context, deps ProviderDeps) error
	// Acquire acquires ammo for shoot. Acquire SHOULD be lightweight, so Instance can Shoot as
	// soon as possible. That means ammo format parsing SHOULD be done in Provider Run goroutine,
	// but acquire just takes ammo from ready queue.
	// Ok false means that shooting MUST be stopped because ammo finished or shooting is canceled.
	// Acquire MAY be called before Run, but SHOULD block until Run is called.
	Acquire() (ammo Ammo, ok bool)
	// Release notifies that ammo usage is finished, and it can be reused.
	// Instance MUST NOT retain references to released ammo.
	Release(ammo Ammo)
}

// ProviderDeps are passed to Provider in Run.
// WARN: another fields could be added in next MINOR versions.
// That is NOT considered as a breaking compatibility change.
type ProviderDeps struct {
	Log    *zap.Logger
	PoolID string
}

//go:generate mockery --name=Gun --case=underscore --outpkg=coremock

// Gun represents logic of making shoots sequentially.
// A Gun is owned by only Instance that uses it for shooting in cycle: Acquire Ammo from Provider ->
// wait for next shoot schedule event -> Shoot with Gun.
// Guns that also implements io.Closer will be Closed after Instance finish.
// Rule of thumb: Guns that create resources which SHOULD be closed after Instance finish,
// SHOULD implement io.Closer.
// Example: Gun that makes HTTP requests through keep alive connection SHOULD close it in Close.
type Gun interface {
	// Bind passes dependencies required for shooting. MUST be called once before any Shoot call.
	Bind(aggr Aggregator, deps GunDeps) error
	// Shoot makes one shoot. Shoot means some abstract load operation: web service or database
	// request, for example.
	// During shoot Gun SHOULD Acquire one or more Samples and Report them to bound Aggregator.
	// Shoot error that MAY mean service under load fail SHOULD be reported to Aggregator in sample
	// and SHOULD be logged to deps.Log at zap.WarnLevel.
	// For example, HTTP request fail SHOULD be Reported and logged,.
	// In case of error, that SHOULD cancel shooting for all Instances Shoot MUST panic using error
	// value describing the problem. That could be configuration error, unsupported Ammo type,
	// situation when service under load doesn't support required protocol,
	Shoot(ammo Ammo)

	// io.Closer // OPTIONAL to implement. See Gun doc for details.
}

// GunDeps are passed to Gun before Instance Run.
// WARN: another fields could be added in next MINOR versions.
// That is NOT considered as a breaking compatibility change.
type GunDeps struct {
	// Ctx is canceled on shoot cancel or finish.
	Ctx context.Context
	// Log fields already contains Id's of Pool and Instance.
	Log *zap.Logger
	// Unique of Gun owning Instance. MAY be used for tagging Samples.
	// Pool set's ids to Instances from 0, incrementing it after Instance Run.
	// There is a race between Instances for Ammo Acquire, so it's not guaranteed, that
	// Instance with lower InstanceId gets it's Ammo earlier.
	InstanceID int
	PoolID     string

	Shared any

	// TODO(skipor): https://github.com/yandex/pandora/issues/71
	// Pass parallelism value. InstanceId MUST be -1 if parallelism > 1.
}

// Sample is data containing shoot report. Return code, timings, shoot meta information.
type Sample interface{}

//go:generate mockery --name=BorrowedSample --case=underscore --outpkg=coremock

// BorrowedSample is Sample that was borrowed from pool, and SHOULD be returned by Aggregator,
// after it will handle Sample.
type BorrowedSample interface {
	Sample
	Return()
}

//go:generate mockery --name=Aggregator --case=underscore --outpkg=coremock

// Aggregator is routine that aggregates Samples from all Pool Instances.
// Usually aggregator is shooting result reporter, that writes Reported Samples
// to DataSink in machine readable format for future analysis.
// An Aggregator MUST be goroutine safe.
// GunDeps are passed to Gun before Instance Run.
type Aggregator interface {
	// Run starts aggregator routine of handling Samples. Blocks until fail or context cancel.
	// Run MUST be called only once. Run SHOULD be called before Report calls, but MAY NOT because
	// of goroutine races.
	// In case of ctx cancel, SHOULD return nil, but MAY ctx.Err(), or error caused ctx.Err()
	// in terms of github.com/pkg/errors.Cause.
	// In case of any dropped Sample (unhandled because of Sample queue overflow) Run SHOULD return
	// error describing how many samples were dropped.
	Run(ctx context.Context, deps AggregatorDeps) error
	// Report reports sample to aggregator. SHOULD be lightweight and not blocking,
	// so Instance can Shoot as soon as possible.
	// That means, that Sample encode and reporting SHOULD NOT be done in caller goroutine,
	// but SHOULD in Aggregator Run goroutine.
	// If Aggregator can't handle Reported Sample without blocking, it SHOULD just drop it.
	// Reported Samples MAY just be dropped, after context cancel.
	// Reported Sample MAY be reused for efficiency, so caller MUST NOT retain reference to Sample.
	// Report MAY be called before Aggregator Run. Report MAY be called after Run finish, in case of
	// Pool Run cancel.
	// Aggregator SHOULD Return Sample if it implements BorrowedSample.
	Report(s Sample)
}

// AggregatorDeps are passed to Aggregator in Run.
// WARN: another fields could be added in next MINOR versions.
// That is NOT considered as a breaking compatibility change.
type AggregatorDeps struct {
	Log *zap.Logger
}

//go:generate mockery --name=Schedule --case=underscore --outpkg=coremock

// Schedule represents operation schedule. Schedule MUST be goroutine safe.
type Schedule interface {
	// Start starts schedule at passed time.
	// Start SHOULD be called once, before any Next call.
	// Start MUST NOT be called more than once or after Next call.
	// If Start was not called, Schedule MUST be started on first Next call.
	Start(startAt time.Time)

	// Next withdraw one operation token and returns next operation time and
	// ok equal true, when Schedule is not finished.
	// If there is no operation tokens left, Next returns Schedule finish time and ok equals false.
	// If Next called first time and Start was not called, Schedule MUST start and return tx
	// equal to start time.
	// Returned ts values MUST increase monotonically. That is, ts returned on next Next call MUST
	// be greater or equal than returned on previous.
	Next() (ts time.Time, ok bool)

	// Left returns n >= 0 number operation token left, if it is known exactly.
	// Returns n < 0, if number of operation tokens is unknown.
	// Left MAY be called before Start.
	Left() int
}

//go:generate mockery --name=DataSource --case=underscore --outpkg=coremock

// DataSource is abstract, ready to only open, source of data.
// Returned source MUST implement io.ReadCloser at least, but can implement more wide interface,
// and this interface methods MAY be used. For example, returned source can be afero.File,
// and can be seeked in such case.
// Examples:
// Dummy os.Stdin wrapper.
// File DataSource that contains filename and afero.Fs, and returns afero.File on OpenSource.
// HTTP DataSource that contains URL and headers used on OpenSource to download content to file,
// and return afero.File, that will be deleted on rc Close.
// String DataSource returns just wrapped *bytes.Buffer with string content.
type DataSource interface {
	// OpenSource opens source for read. OpenSource MUST NOT be called more than once.
	// Returned rc SHOULD have low latency and good enough throughput for Read.
	// rc MAY be afero.File but SHOULD NOT be TCP connection for example.
	// DataSource MAY be some remote resource, but OpenSource SHOULD download all necessary data to
	// local temporary file and return it as rc.
	// Rule of thumb: returned rc SHOULD be afero.File or wrapped *bytes.Buffer.
	// Returned rc SHOULD cleanup all created temporary resources on Close.
	// rc owner SHOULD NOT try cast it to concrete types. For example, rc can be
	// wrapped temporary *os.File, that will be deleted on Close, so it can't be casted to *os.File,
	// because has type of wrapper, but can be used as afero.File.
	// rc Reads SHOULD be buffered for better performance if it doesn't implement io.ByteReader.
	// That usually means that short Reads are efficient enough. It is implemented by *bytes.Buffer
	// and *bufio.Reader, for example. rc Reads MAY be buffered regardless.
	OpenSource() (rc io.ReadCloser, err error)
}

//go:generate mockery --name=DataSink --case=underscore --outpkg=coremock

// DataSink is abstract ready to open sink of data.
//
// Examples:
// Dummy os.Stdout wrapper.
// File DataSink that contains filename and afero.Fs, and returns afero.File on OpenSource.
// HTTP DataSink  caches Written data to temporary file on wc Writes,
// and POST it using contained URL and headers on wc Close.
type DataSink interface {
	// OpenSink opens sink for writing. OpenSink MUST NOT be called more than once.
	// Returned wc SHOULD have low latency and good enough throughput for Write.
	// wc MAY be afero.File but SHOULD NOT be TCP connection for example.
	// DataSink MAY upload Wrote data somewhere but SHOULD do it on wc Close or in background
	// goroutine.
	// wc Writes SHOULD be buffered for better performance if it doesn't implement io.ByteWriter.
	// That usually means that short Writes are efficient enough. It is implemented by *bytes.Buffer
	// and *bufio.Writer, for example. wc Writes MAY be buffered regardless.
	OpenSink() (wc io.WriteCloser, err error)
}
