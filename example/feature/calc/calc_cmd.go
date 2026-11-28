package calc

import (
	"context"
	"errors"
	"math/rand/v2"
	"strconv"
	"sync"

	"github.com/amatsagu/tempest"
)

const (
	initialDisplay  = "0"
	maxCalcSessions = 256
)

type calcSession struct {
	expr    string // holds result after "=" so operations chain cleanly
	display string
}

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
		SlashCommandHandler: c.handle,
	}
}

func (c *calcCommand) handle(ctx context.Context, itx *tempest.CommandInteraction) error {
	session := c.newSession()
	return itx.SendReply(ctx, calcData(session, initialDisplay), false, nil)
}

func (c *calcCommand) handleKey(ctx context.Context, click *tempest.ComponentInteraction) error {
	session := click.PathValue("session")
	key := click.PathValue("key")

	state, ok := c.press(session, key)
	if !ok {
		return click.AcknowledgeWithLinearMessage(ctx, "This calculator session has expired. Run /calc again.", true)
	}

	return click.AcknowledgeWithUpdate(ctx, calcData(session, state.display))
}

func (c *calcCommand) newSession() string {
	id := strconv.FormatUint(rand.Uint64(), 36)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict before inserting so this new session stays intact.
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

// Remove whole runes so multi-byte symbols (×, ÷) aren't split.
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

var errBadExpression = errors.New("bad expression")

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
