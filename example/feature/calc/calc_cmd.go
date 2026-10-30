// Package calc shows an interactive button calculator: /calc posts a keypad and
// each press edits the same message in place.
package calc

import (
	"errors"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amatsagu/tempest"

	"github.com/amatsagu/tempest-example/util"
)

const initialDisplay = "0"

// maxCalcSessions bounds the in-memory session map.
const maxCalcSessions = 256

// calcSession is the state behind one message. After "=" the display keeps the
// equation while expr becomes the result, so presses keep chaining.
type calcSession struct {
	expr    string
	display string
}

// calcCommand owns the live sessions; the session id is packed into each button ID.
type calcCommand struct {
	sessions map[string]calcSession
	mu       sync.Mutex
}

func newCalcCommand() *calcCommand {
	return &calcCommand{sessions: make(map[string]calcSession)}
}

func (c *calcCommand) Definition() tempest.Command {
	return tempest.Command{
		Name:                "calc",
		Description:         "An interactive calculator with buttons.",
		SlashCommandHandler: util.CommandHandler(c.handle),
	}
}

func (c *calcCommand) handle(itx *tempest.CommandInteraction) error {
	session := c.newSession()

	if err := itx.SendReply(calcData(session, initialDisplay), false, nil); err != nil {
		return err
	}

	user := itx.BaseUser()
	ids := calcButtonIDs(session)

	// AwaitComponent auto-acknowledges each press, so the callback edits the
	// original message through the command interaction's webhook token.
	return itx.BaseClient.AwaitComponent(ids, 5*time.Minute, func(click *tempest.ComponentInteraction) bool {
		if user != nil {
			clickUser := click.BaseUser()
			if clickUser == nil || clickUser.ID != user.ID {
				return false
			}
		}

		parts := strings.Split(click.Data.CustomID, ":")
		if len(parts) != 3 {
			return true
		}
		key := parts[2]

		state, ok := c.press(session, key)
		if !ok {
			return true
		}

		if err := itx.EditReply(calcData(session, state.display), false); err != nil {
			return true
		}

		return false
	}, nil)
}

func (c *calcCommand) newSession() string {
	id := strconv.FormatUint(rand.Uint64(), 36)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Make room before inserting, so eviction can never remove this new session.
	for len(c.sessions) >= maxCalcSessions {
		for key := range c.sessions {
			delete(c.sessions, key)
			break
		}
	}

	c.sessions[id] = calcSession{display: initialDisplay}
	return id
}

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

// trimLastRune removes the final rune, not byte, so a multi-byte operator is
// deleted whole.
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

// evaluate replaces the display with "expr = result" and sets expr to the result
// so the next press chains; on a bad expression it shows "error" and leaves expr
// alone.
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

// errBadExpression means the input is malformed or divides by zero.
var errBadExpression = errors.New("bad expression")

// evalExpression evaluates numbers joined by + - × ÷ with × and ÷ precedence, via
// a recursive descent: expr is a chain of terms, a term a chain of numbers.
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
