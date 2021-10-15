package framework

import (
	"golang.org/x/sync/errgroup"
)

// Action is the basic unit of framework
// One http request or websocket connection is handled by multiple Actions
type Action func(*Session) error

// Parallel returns an aggragate Action executing given Actions in parallel
func Parallel(actions ...Action) Action {
	return func(sess *Session) error {
		l := len(actions)
		var g errgroup.Group
		c := make(chan error, l)

		for _, action := range actions {
			action := action
			g.Go(func() error {
				ierr := action(sess)
				c <- ierr
				return ierr
			})
		}

		var err error
		for i := 0; i < l; i++ {
			err = <-c
			if err != nil {
				break
			}
		}

		return err
	}
}

// Seq returns an aggragate Action executing given Actions sequentially
func Seq(actions ...Action) Action {
	return func(sess *Session) error {
		for _, action := range actions {
			err := action(sess)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

// Wrapper adds customs behavior around an Action
// Wrapper is usually used to allocate and release resource for action
// e.g. create & close websocket connection
type Wrapper func(*Session, Action) error

func (p Action) withWrapper(wrapper Wrapper) Action {
	return func(sess *Session) error {
		return wrapper(sess, p)
	}
}

// Wrap returns a new action wrapped by the wrapper
func (p Wrapper) Wrap(action Action) Action {
	return func(sess *Session) error {
		return p(sess, action)
	}
}

// WithWrappers returns an new action with given wrappers
// The first wrapper is most outside
func (p Action) WithWrappers(wrappers ...Wrapper) Action {
	ret := p

	for i := len(wrappers) - 1; i >= 0; i-- {
		// install wrapper in reversed order
		ret = ret.withWrapper(wrappers[i])
	}

	return ret
}
