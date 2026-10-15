// Package calc shows an interactive calculator built from buttons: /calc posts
// a keypad, and each press edits the same message in place.
package calc

import (
	"context"
	"errors"
	"math/rand/v2"
	"strconv"
	"sync"

	"github.com/amatsagu/tempest/discord"
)

// initialDisplay is what an empty expression shows.
const initialDisplay = "0"

// maxCalcSessions bounds the in-memory session map so a burst of /calc calls
// cannot grow it without limit.
const maxCalcSessions = 256

// calcSession is the state behind one message: the expression the buttons
// build, and the line shown. After "=" the display keeps the equation while
// expr becomes the result, so the user can keep chaining.
type calcSession struct {
	expr    string
	display string
}

// calcCommand owns the live calculator sessions. A session id is minted by
// /calc and packed into each button's custom ID, so one registered handler
// serves every session.
type calcCommand struct {
	mu       sync.Mutex
	sessions map[string]calcSession
}

func newCalcCommand() *calcCommand {
	return &calcCommand{sessions: make(map[string]calcSession)}
}

func (c *calcCommand) Definition() discord.Command {
	return discord.Command{
		Name:        "calc",
		Description: "An interactive calculator with buttons.",
	}
}

func (c *calcCommand) handle(_ context.Context, itx *discord.CommandInteraction) error {
	session := c.newSession()
	return itx.Reply(calcData(session, initialDisplay), false)
}

// newSession stores an empty session under a fresh random id.
func (c *calcCommand) newSession() string {
	id := strconv.FormatUint(rand.Uint64(), 36)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.sessions[id] = calcSession{display: initialDisplay}
	c.evict()
	return id
}

// evict drops arbitrary sessions until the map is back under the cap.
//
// ponytail: random eviction, not LRU, and no per-user ownership; a real bot
// would key sessions by user and expire idle ones on a timer.
func (c *calcCommand) evict() {
	for len(c.sessions) > maxCalcSessions {
		for key := range c.sessions {
			delete(c.sessions, key)
			break
		}
	}
}

// press advances one session by a button key and reports whether it existed.
func (c *calcCommand) press(session, key string) (calcSession, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	state, ok := c.sessions[session]
	if !ok {
		return calcSession{}, false
	}
	state = applyKey(state, key)
	c.sessions[session] = state
	return state, true
}

// applyKey mutates a session for one key press.
func applyKey(s calcSession, key string) calcSession {
	switch key {
	case keyClear:
		return calcSession{display: initialDisplay}
	case keyBack:
		s.expr = trimLastRune(s.expr)
		s.display = displayFor(s.expr)
		return s
	case keyEquals:
		return evaluate(s)
	default:
		s.expr += key
		s.display = s.expr
		return s
	}
}

// trimLastRune removes the final rune, not the final byte, so a multi-byte
// operator like × is deleted whole.
func trimLastRune(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	return string(runes[:len(runes)-1])
}

func displayFor(expr string) string {
	if expr == "" {
		return initialDisplay
	}
	return expr
}

// evaluate replaces the display with "expr = result" and the expression with
// the result, so the next press chains off it. On a bad expression the display
// says "error" and the expression is left untouched.
func evaluate(s calcSession) calcSession {
	result, err := evalExpression(s.expr)
	if err != nil {
		s.display = s.expr + " = error"
		return s
	}

	formatted := strconv.FormatFloat(result, 'g', -1, 64)
	s.display = s.expr + " = " + formatted
	s.expr = formatted
	return s
}

// errBadExpression means the input is malformed: a missing operand, a stray
// character, or division by zero.
var errBadExpression = errors.New("bad expression")

// evalExpression evaluates numbers joined by + - × ÷, giving × and ÷
// precedence over + and -. A recursive descent over the runes: expr is a chain
// of terms, a term is a chain of numbers.
func evalExpression(s string) (float64, error) {
	p := parser{runes: []rune(s)}

	value, err := p.expr()
	if err != nil {
		return 0, err
	}
	if p.pos != len(p.runes) {
		return 0, errBadExpression
	}
	return value, nil
}

type parser struct {
	runes []rune
	pos   int
}

func (p *parser) expr() (float64, error) {
	left, err := p.term()
	if err != nil {
		return 0, err
	}
	for p.pos < len(p.runes) && (p.runes[p.pos] == '+' || p.runes[p.pos] == '-') {
		op := p.runes[p.pos]
		p.pos++

		right, err := p.term()
		if err != nil {
			return 0, err
		}
		if op == '+' {
			left += right
		} else {
			left -= right
		}
	}
	return left, nil
}

func (p *parser) term() (float64, error) {
	left, err := p.number()
	if err != nil {
		return 0, err
	}
	for p.pos < len(p.runes) && (p.runes[p.pos] == '×' || p.runes[p.pos] == '÷') {
		op := p.runes[p.pos]
		p.pos++

		right, err := p.number()
		if err != nil {
			return 0, err
		}
		switch op {
		case '×':
			left *= right
		case '÷':
			if right == 0 {
				return 0, errBadExpression
			}
			left /= right
		}
	}
	return left, nil
}

func (p *parser) number() (float64, error) {
	start := p.pos
	dot := false

	for p.pos < len(p.runes) {
		r := p.runes[p.pos]
		if r >= '0' && r <= '9' {
			p.pos++
			continue
		}
		if r == '.' && !dot {
			dot = true
			p.pos++
			continue
		}
		break
	}

	if p.pos == start {
		return 0, errBadExpression
	}
	return strconv.ParseFloat(string(p.runes[start:p.pos]), 64)
}
