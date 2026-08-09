package live

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// A Ctx carries one action from the browser to its handler.
type Ctx struct {
	// Session is the page the action came from.
	Session *Session

	// Action is the name declared on the element with On.
	Action string

	// Element is the id of the element that emitted the event, or empty when
	// it had none.
	Element string

	// Detail is the CustomEvent detail, still encoded. Bind is the usual way
	// to read it; the typed On helper does that for you.
	Detail json.RawMessage

	// Request is the POST that delivered the action, for headers, cookies and
	// the request context.
	Request *http.Request
}

// Context returns the request's context.
func (c *Ctx) Context() context.Context { return c.Request.Context() }

// Bind decodes the event detail into v.
//
// The detail comes from the browser, which means it comes from whoever is
// using the browser. A well-behaved component emits what it says it emits;
// nothing stops a console from emitting something else. Validate what you
// decode.
func (c *Ctx) Bind(v any) error {
	if len(c.Detail) == 0 || string(c.Detail) == "null" {
		return nil
	}
	dec := json.NewDecoder(strings.NewReader(string(c.Detail)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("live: action %q: %w", c.Action, err)
	}
	return nil
}

// Handle returns a handle for the element that emitted the event.
func (c *Ctx) Handle() Handle { return c.Session.Element(c.Element) }

// A Handler runs one action.
//
// Returning an error logs it and answers 500. It does not reach the browser:
// what the user should see is a patch, sent from the handler, in whatever
// terms the page understands.
type Handler func(*Ctx) error

// On registers a server-wide handler for an action.
func (s *Server) On(action string, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.actions[action] = h
}

// On registers a handler for this session only, taking precedence over the
// server-wide one. Use it to close over per-page state.
func (s *Session) On(action string, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.actions == nil {
		s.actions = map[string]Handler{}
	}
	s.actions[action] = h
}

// On registers a handler that receives the event detail already decoded into
// T, which is what the generated detail types are for:
//
//	live.On(srv, "add-todo", func(c *live.Ctx, d ui.TodoListAddDetail) error {
//	    ...
//	})
func On[T any](s *Server, action string, h func(*Ctx, T) error) {
	s.On(action, bind(h))
}

// OnSession is On for a single session.
func OnSession[T any](s *Session, action string, h func(*Ctx, T) error) {
	s.On(action, bind(h))
}

func bind[T any](h func(*Ctx, T) error) Handler {
	return func(c *Ctx) error {
		var detail T
		if err := c.Bind(&detail); err != nil {
			return err
		}
		return h(c, detail)
	}
}

func (s *Session) handlerFor(action string) (Handler, bool) {
	s.mu.Lock()
	h, ok := s.actions[action]
	s.mu.Unlock()
	if ok {
		return h, true
	}
	s.srv.mu.RLock()
	defer s.srv.mu.RUnlock()
	h, ok = s.srv.actions[action]
	return h, ok
}

// actionRequest is what the client posts.
type actionRequest struct {
	Session string          `json:"s"`
	Action  string          `json:"a"`
	Element string          `json:"i"`
	Detail  json.RawMessage `json:"d"`
}

func (s *Server) serveAction(w http.ResponseWriter, r *http.Request) {
	if !s.originAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	body := http.MaxBytesReader(w, r.Body, s.opts.MaxDetail)
	defer body.Close()

	var req actionRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	sess, ok := s.Session(req.Session)
	if !ok {
		// The session expired, or the id is wrong. Either way this page can no
		// longer act, and the app should decide what to do about it.
		http.Error(w, "unknown session", http.StatusNotFound)
		return
	}

	handler, ok := sess.handlerFor(req.Action)
	if !ok {
		// An element declared an action nobody handles: a rename that only
		// happened on one side. Silence would make that impossible to find.
		s.opts.Logger.Warn("alacris live: no handler for action",
			"action", req.Action, "element", req.Element)
		http.Error(w, "unknown action", http.StatusNotFound)
		return
	}

	ctx := &Ctx{
		Session: sess,
		Action:  req.Action,
		Element: req.Element,
		Detail:  req.Detail,
		Request: r,
	}
	if err := handler(ctx); err != nil {
		s.opts.Logger.Error("alacris live: action failed",
			"action", req.Action, "element", req.Element, "error", err)
		http.Error(w, "action failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// originAllowed keeps a cross-site page from driving a session.
//
// The session id is already secret, so this is defence in depth rather than
// the only thing standing in the way. A missing Origin is allowed: same-origin
// GETs and some clients omit it, and rejecting those breaks more than it
// protects.
func (s *Server) originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if s.opts.AllowOrigin != nil {
		return s.opts.AllowOrigin(origin, r)
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}
