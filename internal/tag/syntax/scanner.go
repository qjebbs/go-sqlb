package syntax

// scanFn is the lexical scan function
type scanFn func(*scanner) scanFn

// scanner is the lexical scanner
type scanner struct {
	*lexerHelper

	tokens []*token
	token  *token
	state  scanFn
}

func newScanner(input string) *scanner {
	s := &scanner{
		lexerHelper: newLexerHelper(input),
		state:       scan,
	}
	return s
}

func (s *scanner) emitToken(t TokenType, bad bool) {
	s.tokens = append(s.tokens, &token{
		typ:   t,
		bad:   bad,
		start: s.start.offset,
		end:   s.current.offset,
		pos:   s.start.Pos,
		lit:   s.input[s.start.offset:s.current.offset],
	})
}

// NextToken finds the next token
func (s *scanner) NextToken() bool {
	for (len(s.tokens) == 0) && s.state != nil {
		s.state = s.state(s)
	}
	if len(s.tokens) > 0 {
		s.token = s.tokens[0]
		s.tokens = s.tokens[1:]
		return true
	}
	return false
}

func scan(s *scanner) scanFn {
	s.SkipWhitespace()
	s.StartToken()
	if s.IsLetter() {
		return scanKey
	}
	switch r := s.rune; r {
	case EOF:
		s.emitToken(_EOF, false)
		return nil
	default:
		s.Next()
		s.emitToken(_Error, true)
		return nil
	}
}

func scanKey(s *scanner) scanFn {
	s.SkipWhitespace()
	s.StartToken()
	for s.IsLetter() {
		s.Next()
	}
	s.emitToken(_Key, false)
	return scanAfterKey
}

func scanAfterKey(s *scanner) scanFn {
	s.SkipWhitespace()
	s.StartToken()
	switch s.rune {
	case ':':
		s.Next()
		s.emitToken(_Colon, false)
		return scanValue
	case ';':
		s.Next()
		s.emitToken(_Semicolon, false)
		return scan
	case EOF:
		s.emitToken(_EOF, false)
		return nil
	default:
		return scanValue
	}
}

func scanValue(s *scanner) scanFn {
	s.SkipWhitespace()
	s.StartToken()
	for r := s.rune; r != ';' && r != EOF; r = s.Next() {
	}
	s.TrimTrailingWhitespace()
	if s.Advanced() {
		s.emitToken(_Value, false)
	}
	return scanAfterValue
}

func scanAfterValue(s *scanner) scanFn {
	s.SkipWhitespace()
	s.StartToken()
	switch s.rune {
	case ';':
		s.Next()
		s.emitToken(_Semicolon, false)
		return scan
	case EOF:
		s.emitToken(_EOF, false)
		return nil
	default:
		return scan
	}
}
