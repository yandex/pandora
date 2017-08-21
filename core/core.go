// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

// package core defines pandora engine extension points.
// Core interfaces implementations can be used for manual engine creation and using as a library,
// or can be registered in pandora plugin system (look at core/plugin pkg), for creating engine from abstract config.
package core

import (
	"context"
	"time"
)

// TODO(skipor): pass some dependencies from engine to core extpoints.
// Extra argument to Run, or through context values, maybe.

// Ammo is data required for one shot. Usually it contains something that differs
// from one shot to another.
// Something like requested recourse indetificator, query params, meta information
// helpful for future shooting analysis.
type Ammo interface{}

//go:generate mockery -name=Provider -case=underscore -outpkg=coremock

// Provider is routine that generates ammo for instance shoots.
// A Provider must be goroutine safe.
type Provider interface {
	// Run starts provider routine. Blocks until ammo finish, error or context cancel.
	// Run must be called once before any Acquire or Release calls.
	// In case of context cancel, return nil (recommended), ctx.Err(), or error caused ctx.Err()
	// in terms of github.com/pkg/errors.Cause.
	Run(context.Context) error
	// Acquire acquires ammo for shoot. Should be lightweight, so instance can shoot as
	// soon as possible. That means ammo format parsing done in provider background routine,
	// and acquire just takes ammo from ready pool.
	// Ok false means that shooting should be stopped: ammo finished or shooting is canceled.
	// Acquire may be called before start, but may block until start is called.
	Acquire() (a Ammo, ok bool)
	// Release notifies that ammo usage is finished, and it can be reused.
	// Instance should not retain references to released ammo.
	Release(Ammo)
}

// Sample is data containing shoot report. Return code, timings, shoot meta information.
type Sample interface{}

//go:generate mockery -name=Aggregator -case=underscore -outpkg=coremock

// Aggregator is routine that aggregates samples from all instances.
// Usually aggregator is shooting result reporter, that writes released samples
// to file in machine readable format for future analysis.
// An Aggregator must be goroutine safe.
type Aggregator interface {
	// Run starts aggregator routine. Blocks until error or context cancel.
	// In case of context cancel, return nil, ctx.Err(), or error caused ctx.Err()
	// in terms of github.com/pkg/errors.Cause in case of successful run, or other error
	// if failed.
	Run(context.Context) error
	// Report reports sample to aggregator. Should be lightweight and not blocking,
	// so instance can shoot as soon as possible.
	// That means, that sample encode and reporting IO done in aggregator provider routine.
	// If Aggregator can't process reported sample without blocking, it should just throw it away.
	// If any reported samples were thrown away, Run should return error describing how many samples
	// were thrown away.
	// Reported sample can be reused for efficiency.
	// Report may be called before Aggregator Run.
	Report(Sample)
}

//go:generate mockery -name=Schedule -case=underscore -outpkg=coremock

// Schedule represents operation schedule. Schedule must be goroutine safe.
type Schedule interface {
	// Run starts schedule at passed time.
	// Run may be called once, before any Next call. (Before, means not concurrently too.)
	// If start was not called, schedule is started at first Next call.
	Start(startAt time.Time)
	// Next withdraw one operation token and returns next operation time and
	// ok equal true, when schedule is not finished.
	// If there is no operation tokens left, Next returns Schedule
	// finish time and ok equals false.
	Next() (ts time.Time, ok bool)
}

//go:generate mockery -name=Gun -case=underscore -outpkg=coremock

// Gun represents logic of making shoots sequentially.
// A Gun is owned by only instance that uses it for shooting in cycle: acquire ammo from provider ->
// wait for next shoot schedule event -> shoot with gun.
// Guns that also implements io.Closer will be closed after instance finish.
// Actually, Guns that create resources which should be closed after instance finish,
// SHOULD also implement io.Closer
type Gun interface {
	// TODO(skipor): shoot context is same for now. Maybe pass it to Bind instead?

	// Bind passes dependencies required for shooting. Called once before shooting start.
	Bind(Aggregator)
	// Shoot makes one shoot. Shoot means some abstract load operation: web service or database request, for example.
	// During shoot Gun acquires one or more samples and report them to bound Aggregator.
	// Shoot error should be reported to Aggregator in sample, and logged maybe.
	// In case of error, that should cancel shooting for all instances (configuration problem
	// or unexpected behaviour for example) Shoot should panic with error value.
	// http.Request fail is not error for panic, but error for reporting to aggregator.
	Shoot(context.Context, Ammo)

	// io.Closer // Optional. See Gun doc for details.
}
