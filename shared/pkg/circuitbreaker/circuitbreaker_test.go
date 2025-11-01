package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_Execute(t *testing.T) {
	t.Run("success execution", func(t *testing.T) {
		cb := New("test", Config{
			MaxRequests: 2,
			Interval:    time.Second,
			Timeout:     time.Second,
		})

		err := cb.Execute(func() error {
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if cb.State() != StateClosed {
			t.Errorf("expected closed state, got %v", cb.State())
		}
	})

	t.Run("failure execution", func(t *testing.T) {
		cb := New("test", Config{
			MaxRequests: 2,
			Interval:    time.Second,
			Timeout:     time.Second,
			ReadyToTrip: func(counts Counts) bool {
				return counts.ConsecutiveFailures > 2
			},
		})

		// Fail 3 times to open the circuit
		for i := 0; i < 3; i++ {
			cb.Execute(func() error {
				return errors.New("test error")
			})
		}

		if cb.State() != StateOpen {
			t.Errorf("expected open state, got %v", cb.State())
		}

		// Next request should fail immediately
		err := cb.Execute(func() error {
			return nil
		})

		if !errors.Is(err, ErrCircuitOpen) {
			t.Errorf("expected circuit open error, got %v", err)
		}
	})

	t.Run("half-open to closed transition", func(t *testing.T) {
		cb := New("test", Config{
			MaxRequests: 2,
			Interval:    time.Second,
			Timeout:     100 * time.Millisecond,
			ReadyToTrip: func(counts Counts) bool {
				return counts.ConsecutiveFailures > 2
			},
		})

		// Trip the circuit
		for i := 0; i < 3; i++ {
			cb.Execute(func() error {
				return errors.New("test error")
			})
		}

		if cb.State() != StateOpen {
			t.Fatalf("expected open state, got %v", cb.State())
		}

		// Wait for timeout to transition to half-open
		time.Sleep(150 * time.Millisecond)

		// Successful requests should close the circuit
		for i := 0; i < 2; i++ {
			err := cb.Execute(func() error {
				return nil
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		if cb.State() != StateClosed {
			t.Errorf("expected closed state, got %v", cb.State())
		}
	})
}

func TestCircuitBreaker_ExecuteWithContext(t *testing.T) {
	t.Run("successful execution with context", func(t *testing.T) {
		cb := New("test", Config{
			MaxRequests: 2,
			Interval:    time.Second,
			Timeout:     time.Second,
		})

		ctx := context.Background()
		err := cb.ExecuteWithContext(ctx, func(ctx context.Context) error {
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("context timeout", func(t *testing.T) {
		cb := New("test", Config{
			MaxRequests: 2,
			Interval:    time.Second,
			Timeout:     time.Second,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := cb.ExecuteWithContext(ctx, func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})

		if !errors.Is(err, ErrTimeout) {
			t.Errorf("expected timeout error, got %v", err)
		}
	})
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	stateChanges := make([]State, 0)
	cb := New("test", Config{
		MaxRequests: 1,
		Timeout:     100 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures > 1
		},
		OnStateChange: func(name string, from State, to State) {
			stateChanges = append(stateChanges, to)
		},
	})

	// Fail to open
	cb.Execute(func() error { return errors.New("error") })
	cb.Execute(func() error { return errors.New("error") })

	if cb.State() != StateOpen {
		t.Errorf("expected open state, got %v", cb.State())
	}

	// Wait for half-open
	time.Sleep(150 * time.Millisecond)
	cb.Execute(func() error { return nil })

	if cb.State() != StateClosed {
		t.Errorf("expected closed state, got %v", cb.State())
	}

	if len(stateChanges) < 2 {
		t.Errorf("expected at least 2 state changes, got %d", len(stateChanges))
	}
}

func TestCircuitBreaker_Counts(t *testing.T) {
	cb := New("test", Config{
		MaxRequests: 2,
		Interval:    time.Second,
		Timeout:     time.Second,
	})

	// Success
	cb.Execute(func() error { return nil })
	counts := cb.Counts()

	if counts.TotalSuccesses != 1 {
		t.Errorf("expected 1 success, got %d", counts.TotalSuccesses)
	}

	// Failure
	cb.Execute(func() error { return errors.New("error") })
	counts = cb.Counts()

	if counts.TotalFailures != 1 {
		t.Errorf("expected 1 failure, got %d", counts.TotalFailures)
	}

	if counts.ConsecutiveFailures != 1 {
		t.Errorf("expected 1 consecutive failure, got %d", counts.ConsecutiveFailures)
	}

	if counts.ConsecutiveSuccesses != 0 {
		t.Errorf("expected 0 consecutive successes, got %d", counts.ConsecutiveSuccesses)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := New("test", Config{
		MaxRequests: 2,
		Timeout:     time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures > 1
		},
	})

	// Trip the circuit
	cb.Execute(func() error { return errors.New("error") })
	cb.Execute(func() error { return errors.New("error") })

	if cb.State() != StateOpen {
		t.Fatalf("expected open state, got %v", cb.State())
	}

	// Reset
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("expected closed state after reset, got %v", cb.State())
	}

	counts := cb.Counts()
	if counts.TotalFailures != 0 {
		t.Errorf("expected 0 failures after reset, got %d", counts.TotalFailures)
	}
}

func TestCircuitBreakerManager(t *testing.T) {
	manager := NewManager(Config{
		MaxRequests: 2,
		Timeout:     time.Second,
	})

	t.Run("get creates new breaker", func(t *testing.T) {
		cb1 := manager.Get("service1")
		if cb1 == nil {
			t.Fatal("expected circuit breaker, got nil")
		}

		if cb1.Name() != "service1" {
			t.Errorf("expected name 'service1', got '%s'", cb1.Name())
		}
	})

	t.Run("get returns existing breaker", func(t *testing.T) {
		cb1 := manager.Get("service2")
		cb2 := manager.Get("service2")

		if cb1 != cb2 {
			t.Error("expected same circuit breaker instance")
		}
	})

	t.Run("remove deletes breaker", func(t *testing.T) {
		cb1 := manager.Get("service3")
		manager.Remove("service3")
		cb2 := manager.Get("service3")

		if cb1 == cb2 {
			t.Error("expected different circuit breaker instance after removal")
		}
	})

	t.Run("get all returns all breakers", func(t *testing.T) {
		manager.Get("service4")
		manager.Get("service5")

		all := manager.GetAll()
		if len(all) < 2 {
			t.Errorf("expected at least 2 breakers, got %d", len(all))
		}
	})

	t.Run("reset resets all breakers", func(t *testing.T) {
		cb := manager.Get("service6")

		// Trip the circuit
		cb.Execute(func() error { return errors.New("error") })
		cb.Execute(func() error { return errors.New("error") })
		cb.Execute(func() error { return errors.New("error") })

		manager.Reset()

		if cb.State() != StateClosed {
			t.Errorf("expected closed state after reset, got %v", cb.State())
		}
	})
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("State.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
