package analysis

type PorterStemmer struct{}

func NewPorterStemmer() *PorterStemmer {
	return &PorterStemmer{}
}

func (s *PorterStemmer) Filter(tokens []string) []string {
	out := make([]string, len(tokens))
	for i, t := range tokens {
		out[i] = stem(t)
	}
	return out
}

func stem(word string) string {
	if len(word) <= 2 {
		return word
	}
	for i := 0; i < len(word); i++ {
		if word[i] < 'a' || word[i] > 'z' {
			return word
		}
	}

	p := &porter{b: []byte(word), k: len(word) - 1}
	p.step1ab()
	if p.k > 0 {
		p.step1c()
		p.step2()
		p.step3()
		p.step4()
		p.step5()
	}
	return string(p.b[:p.k+1])
}

type porter struct {
	b []byte
	k int
	j int
}

func (p *porter) cons(i int) bool {
	switch p.b[i] {
	case 'a', 'e', 'i', 'o', 'u':
		return false
	case 'y':
		if i == 0 {
			return true
		}
		return !p.cons(i - 1)
	}
	return true
}

func (p *porter) m() int {
	n, i := 0, 0
	for {
		if i > p.j {
			return n
		}
		if !p.cons(i) {
			break
		}
		i++
	}
	i++
	for {
		for {
			if i > p.j {
				return n
			}
			if p.cons(i) {
				break
			}
			i++
		}
		i++
		n++
		for {
			if i > p.j {
				return n
			}
			if !p.cons(i) {
				break
			}
			i++
		}
		i++
	}
}

func (p *porter) vowelinstem() bool {
	for i := 0; i <= p.j; i++ {
		if !p.cons(i) {
			return true
		}
	}
	return false
}

func (p *porter) doublec(j int) bool {
	if j < 1 || p.b[j] != p.b[j-1] {
		return false
	}
	return p.cons(j)
}

func (p *porter) cvc(i int) bool {
	if i < 2 || !p.cons(i) || p.cons(i-1) || !p.cons(i-2) {
		return false
	}
	switch p.b[i] {
	case 'w', 'x', 'y':
		return false
	}
	return true
}

func (p *porter) ends(s string) bool {
	l := len(s)
	if l > p.k+1 {
		return false
	}
	if string(p.b[p.k-l+1:p.k+1]) != s {
		return false
	}
	p.j = p.k - l
	return true
}

func (p *porter) setto(s string) {
	p.b = append(p.b[:p.j+1], []byte(s)...)
	p.k = len(p.b) - 1
}

func (p *porter) r(s string) {
	if p.m() > 0 {
		p.setto(s)
	}
}

func (p *porter) step1ab() {
	if p.b[p.k] == 's' {
		switch {
		case p.ends("sses"):
			p.k -= 2
		case p.ends("ies"):
			p.setto("i")
		case p.b[p.k-1] != 's':
			p.k--
		}
	}
	if p.ends("eed") {
		if p.m() > 0 {
			p.k--
		}
	} else if (p.ends("ed") || p.ends("ing")) && p.vowelinstem() {
		p.k = p.j
		switch {
		case p.ends("at"):
			p.setto("ate")
		case p.ends("bl"):
			p.setto("ble")
		case p.ends("iz"):
			p.setto("ize")
		case p.doublec(p.k):
			p.k--
			switch p.b[p.k] {
			case 'l', 's', 'z':
				p.k++
			}
		case p.m() == 1 && p.cvc(p.k):
			p.setto("e")
		}
	}
}

func (p *porter) step1c() {
	if p.ends("y") && p.vowelinstem() {
		p.b[p.k] = 'i'
	}
}

func (p *porter) step2() {
	if p.k == 0 {
		return
	}
	switch p.b[p.k-1] {
	case 'a':
		switch {
		case p.ends("ational"):
			p.r("ate")
		case p.ends("tional"):
			p.r("tion")
		}
	case 'c':
		switch {
		case p.ends("enci"):
			p.r("ence")
		case p.ends("anci"):
			p.r("ance")
		}
	case 'e':
		if p.ends("izer") {
			p.r("ize")
		}
	case 'l':
		switch {
		case p.ends("bli"):
			p.r("ble")
		case p.ends("alli"):
			p.r("al")
		case p.ends("entli"):
			p.r("ent")
		case p.ends("eli"):
			p.r("e")
		case p.ends("ousli"):
			p.r("ous")
		}
	case 'o':
		switch {
		case p.ends("ization"):
			p.r("ize")
		case p.ends("ation"):
			p.r("ate")
		case p.ends("ator"):
			p.r("ate")
		}
	case 's':
		switch {
		case p.ends("alism"):
			p.r("al")
		case p.ends("iveness"):
			p.r("ive")
		case p.ends("fulness"):
			p.r("ful")
		case p.ends("ousness"):
			p.r("ous")
		}
	case 't':
		switch {
		case p.ends("aliti"):
			p.r("al")
		case p.ends("iviti"):
			p.r("ive")
		case p.ends("biliti"):
			p.r("ble")
		}
	case 'g':
		if p.ends("logi") {
			p.r("log")
		}
	}
}

func (p *porter) step3() {
	switch p.b[p.k] {
	case 'e':
		switch {
		case p.ends("icate"):
			p.r("ic")
		case p.ends("ative"):
			p.r("")
		case p.ends("alize"):
			p.r("al")
		}
	case 'i':
		if p.ends("iciti") {
			p.r("ic")
		}
	case 'l':
		switch {
		case p.ends("ical"):
			p.r("ic")
		case p.ends("ful"):
			p.r("")
		}
	case 's':
		if p.ends("ness") {
			p.r("")
		}
	}
}

func (p *porter) step4() {
	if p.k == 0 {
		return
	}
	switch p.b[p.k-1] {
	case 'a':
		if !p.ends("al") {
			return
		}
	case 'c':
		if !p.ends("ance") && !p.ends("ence") {
			return
		}
	case 'e':
		if !p.ends("er") {
			return
		}
	case 'i':
		if !p.ends("ic") {
			return
		}
	case 'l':
		if !p.ends("able") && !p.ends("ible") {
			return
		}
	case 'n':
		if !p.ends("ant") && !p.ends("ement") && !p.ends("ment") && !p.ends("ent") {
			return
		}
	case 'o':
		if !((p.ends("ion") && p.j >= 0 && (p.b[p.j] == 's' || p.b[p.j] == 't')) || p.ends("ou")) {
			return
		}
	case 's':
		if !p.ends("ism") {
			return
		}
	case 't':
		if !p.ends("ate") && !p.ends("iti") {
			return
		}
	case 'u':
		if !p.ends("ous") {
			return
		}
	case 'v':
		if !p.ends("ive") {
			return
		}
	case 'z':
		if !p.ends("ize") {
			return
		}
	default:
		return
	}
	if p.m() > 1 {
		p.k = p.j
	}
}

func (p *porter) step5() {
	p.j = p.k
	if p.b[p.k] == 'e' {
		a := p.m()
		if a > 1 || (a == 1 && !p.cvc(p.k-1)) {
			p.k--
		}
	}
	if p.b[p.k] == 'l' && p.doublec(p.k) && p.m() > 1 {
		p.k--
	}
}
