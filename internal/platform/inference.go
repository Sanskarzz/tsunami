package platform

import (
	"context"
	"errors"
	"time"

	"github.com/Sanskarzz/tsunami/internal/inference"
	"github.com/stacklok/mecatl/engine/session"
)

func (s *Server) Infer(parent context.Context, id string, in inference.Request) (inference.Reply, error) {
	if s.Inference == nil {
		return inference.Reply{}, errors.New("model disabled")
	}
	if err := inference.ValidateRequest(in); err != nil {
		return inference.Reply{}, err
	}
	ctx, cancel, op, err := s.beginFor(id, 20*time.Second)
	if err != nil {
		return inference.Reply{}, err
	}
	stopParent := context.AfterFunc(parent, cancel)
	defer func() { stopParent(); cancel(); s.mu.Lock(); delete(s.active[id], op); s.mu.Unlock() }()
	s.mu.Lock()
	if s.modelActive[id] || s.modelInFlight >= 2 {
		s.mu.Unlock()
		return inference.Reply{}, errors.New("model concurrency limit")
	}
	var observation *inference.Observation
	err = s.Store.Update(id, func(r *Run) error {
		if r.State != Running || time.Now().After(r.Deadline) || r.Execution != "live" || r.Model == nil || r.Model.ConfigID != s.Inference.Identity() {
			return errors.New("model permission revoked or configuration changed")
		}
		if r.Model.CallsReserved >= r.Model.MaxCalls {
			return errors.New("model call quota exhausted")
		}
		if r.Assessment != nil && (!r.Assessment.Canonical() || r.Assessment.Digest() != r.AssessmentDigest) {
			return errors.New("assessment policy changed")
		}
		r.Model.CallsReserved++
		r.Model.UsageUncertain = true
		deadline, _ := ctx.Deadline()
		observation = inference.NewObservation(r.Model.CallsReserved, deadline)
		observation.Record.DeadlineSource = "call"
		if deadline.Equal(r.Deadline) {
			observation.Record.DeadlineSource = "run"
		}
		r.Model.Calls = append(r.Model.Calls, observation.Record)
		return nil
	})
	if err != nil {
		s.mu.Unlock()
		return inference.Reply{}, err
	}
	if s.modelActive == nil {
		s.modelActive = map[string]bool{}
	}
	s.modelActive[id] = true
	s.modelInFlight++
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.modelActive, id); s.modelInFlight--; s.mu.Unlock() }()
	r, err := s.Store.Get(id)
	if err != nil {
		return inference.Reply{}, err
	}
	reserve := func(ctx context.Context) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.Store.Update(id, func(r *Run) error {
			if r.State != Running || time.Now().After(r.Deadline) || r.Model == nil {
				return errors.New("model permission revoked")
			}
			if r.Model.WireAttempts >= r.Model.MaxCalls {
				return errors.New("upstream attempt quota exhausted")
			}
			r.Model.WireAttempts++
			return nil
		})
	}
	opened := func(ctx context.Context, status int) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if status != 200 {
			return nil
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.Store.Update(id, func(r *Run) error {
			if r.State != Running || r.Model == nil || time.Now().After(r.Deadline) {
				return errors.New("model permission revoked")
			}
			r.Model.StreamsOpened++
			for i := range r.Model.Calls {
				if r.Model.Calls[i].Sequence == observation.Record.Sequence {
					r.Model.Calls[i] = observation.Record
					break
				}
			}
			return nil
		})
	}
	var reply inference.Reply
	var usage *session.Usage
	var callErr error
	if r.Assessment != nil {
		if !r.Assessment.Canonical() || r.Assessment.Digest() != r.AssessmentDigest {
			return inference.Reply{}, errors.New("assessment policy changed")
		}
		reply, usage, callErr = s.Inference.CompleteObserved(ctx, r.Target, in, r.Assessment, observation, reserve, opened)
	} else {
		reply, usage, callErr = s.Inference.CompleteObserved(ctx, r.Target, in, nil, observation, reserve, opened)
	}
	if contextError := inference.ContextFailure(ctx); contextError != nil {
		callErr = contextError
		observation.Finish(callErr)
	}
	// Missing usage or a failed stream leaves its reservation spent. A later
	// successful call cannot erase uncertainty from an earlier failed attempt.
	err = s.Store.Update(id, func(r *Run) error {
		if r.Model == nil {
			return errors.New("model state missing")
		}
		for i := range r.Model.Calls {
			if r.Model.Calls[i].Sequence == observation.Record.Sequence {
				r.Model.Calls[i] = observation.Record
				break
			}
		}
		if callErr == nil {
			r.Model.CompletedCalls++
			if usage != nil {
				r.Model.InputTokens += usage.InputTokens
				r.Model.OutputTokens += usage.OutputTokens
			}
		}
		if usage == nil || callErr != nil {
			r.Events = append(r.Events, Event{time.Now().UTC(), "model_usage_uncertain", "call reservation retained"})
		}
		if callErr != nil {
			r.Events = append(r.Events, Event{time.Now().UTC(), "model_error", inference.ErrorCode(callErr)})
		}
		// Only fully-accounted calls in the entire run can clear uncertainty.
		r.Model.UsageUncertain = r.Model.CompletedCalls != r.Model.CallsReserved || r.Model.WireAttempts != r.Model.CompletedCalls
		for _, event := range r.Events {
			if event.Kind == "model_usage_uncertain" {
				r.Model.UsageUncertain = true
				break
			}
		}
		return nil
	})
	if err != nil {
		return inference.Reply{}, err
	}
	if callErr != nil || ctx.Err() != nil {
		return inference.Reply{}, errors.New("model call failed or cancelled")
	}
	return reply, nil
}
