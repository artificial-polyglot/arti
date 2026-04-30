package read

type StylePair struct {
	Style    string
	StyleNum string
}

type USFMStack struct {
	items []StylePair
}

func (s *USFMStack) Push(style string, styleNum string) {
	s.items = append(s.items, StylePair{style, styleNum})
}

func (s *USFMStack) Pop() (StylePair, bool) {
	if len(s.items) == 0 {
		return StylePair{}, false
	}
	top := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return top, true
}

func (s *USFMStack) Peek() (StylePair, bool) {
	if len(s.items) == 0 {
		return StylePair{}, false
	}
	return s.items[len(s.items)-1], true
}

func (s *USFMStack) IsEmpty() bool {
	return len(s.items) == 0
}

func (s *USFMStack) Len() int {
	return len(s.items)
}
