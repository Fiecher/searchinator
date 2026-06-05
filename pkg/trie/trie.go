package trie

type Trie struct {
	root *trieNode
	size int
}

type trieNode struct {
	children map[rune]*trieNode
	terminal bool
}

func newNode() *trieNode {
	return &trieNode{children: make(map[rune]*trieNode)}
}

func New() *Trie {
	return &Trie{root: newNode()}
}

func (t *Trie) Insert(word string) {
	n := t.root
	for _, r := range word {
		child, ok := n.children[r]
		if !ok {
			child = newNode()
			n.children[r] = child
		}
		n = child
	}
	if !n.terminal {
		n.terminal = true
		t.size++
	}
}

func (t *Trie) Len() int { return t.size }

func (t *Trie) WithPrefix(prefix string) []string {
	n := t.root
	for _, r := range prefix {
		child, ok := n.children[r]
		if !ok {
			return nil
		}
		n = child
	}
	var out []string
	collect(n, prefix, &out)
	return out
}

func (t *Trie) Contains(word string) bool {
	n := t.root
	for _, r := range word {
		child, ok := n.children[r]
		if !ok {
			return false
		}
		n = child
	}
	return n.terminal
}

func collect(n *trieNode, prefix string, out *[]string) {
	if n.terminal {
		*out = append(*out, prefix)
	}

	keys := make([]rune, 0, len(n.children))
	for r := range n.children {
		keys = append(keys, r)
	}
	sortRunes(keys)
	for _, r := range keys {
		collect(n.children[r], prefix+string(r), out)
	}
}

func sortRunes(rs []rune) {
	for i := 1; i < len(rs); i++ {
		for j := i; j > 0 && rs[j] < rs[j-1]; j-- {
			rs[j], rs[j-1] = rs[j-1], rs[j]
		}
	}
}
