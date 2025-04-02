package cache

import (
	"strings"
	"sync"
	"time"
)

type TrieNode struct {
	children map[rune]*TrieNode
	isEnd    bool
}

type timerEntry struct {
	timer      *time.Timer
	generation int
}

type Trie struct {
	root     *TrieNode
	mutex    sync.RWMutex
	duration time.Duration
	timers   map[string]*timerEntry // Regular map with mutex protection
}

func NewTrie(dd time.Duration) *Trie {
	return &Trie{
		root:     &TrieNode{children: make(map[rune]*TrieNode)},
		duration: dd,
		timers:   make(map[string]*timerEntry),
	}
}

func (t *Trie) Insert(word string, duration time.Duration) bool {
	if word == "" {
		return false
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.insertWord(word, duration)
}

func (t *Trie) InsertConditional(prefix, suffix string, duration time.Duration, maxCount int) bool {
	if prefix == "" || suffix == "" {
		return false
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	if len(t.getWordsWithPrefix(prefix)) >= maxCount {
		return false
	}

	return t.insertWord(prefix+suffix, duration)
}

func (t *Trie) InsertConditionalDefaultDuration(prefix, suffix string, maxCount int) bool {
	return t.InsertConditional(prefix, suffix, t.duration, maxCount)
}

func (t *Trie) insertWord(word string, duration time.Duration) bool {
	if entry, exists := t.timers[word]; exists {
		entry.generation++
		entry.timer.Reset(duration)
		return true
	}

	node := t.root
	for _, ch := range word {
		if _, exists := node.children[ch]; !exists {
			node.children[ch] = &TrieNode{children: make(map[rune]*TrieNode)}
		}
		node = node.children[ch]
	}
	node.isEnd = true

	generation := 0
	entry := &timerEntry{
		timer: time.AfterFunc(duration, func() {
			t.mutex.Lock()
			defer t.mutex.Unlock()

			currentEntry, exists := t.timers[word]
			if exists && currentEntry.generation == generation {
				delete(t.timers, word)
				t.deleteWord(t.root, word, 0)
			}
		}),
		generation: generation,
	}
	t.timers[word] = entry

	return true
}

func (t *Trie) Search(word string, isPrefix bool) bool {
	if word == "" {
		return false
	}

	t.mutex.RLock()
	defer t.mutex.RUnlock()

	node := t.root
	for _, ch := range word {
		if _, exists := node.children[ch]; !exists {
			return false
		}
		node = node.children[ch]
	}

	return isPrefix || node.isEnd
}

func (t *Trie) GetWords() []byte {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	var sb strings.Builder
	for k, _ := range t.timers {
		sb.WriteString(k + "\r\n")
	}
	return []byte(sb.String())
}

func (t *Trie) GetWordsWithPrefix(prefix string) []string {
	if prefix == "" {
		return nil
	}

	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.getWordsWithPrefix(prefix)
}

func (t *Trie) getWordsWithPrefix(prefix string) []string {
	node := t.root
	for _, ch := range prefix {
		if _, exists := node.children[ch]; !exists {
			return nil
		}
		node = node.children[ch]
	}
	var words []string
	t.collectWords(node, prefix, &words)
	return words
}

func (t *Trie) collectWords(node *TrieNode, prefix string, words *[]string) {
	if node.isEnd {
		*words = append(*words, prefix)
	}
	for ch, child := range node.children {
		t.collectWords(child, prefix+string(ch), words)
	}
}

func (t *Trie) DeleteSingle(prefix, suffix string) bool {
	return t.Delete(prefix+suffix, false)
}

func (t *Trie) Delete(word string, isPrefix bool) bool {
	if word == "" {
		return false
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	if isPrefix {
		t.deletePrefix(word)
	} else {
		if entry, exists := t.timers[word]; exists {
			entry.timer.Stop()
			delete(t.timers, word)
			t.deleteWord(t.root, word, 0)
		}
	}

	return true
}

func (t *Trie) deleteWord(node *TrieNode, word string, index int) bool {
	if index == len(word) {
		if !node.isEnd {
			return false
		}
		node.isEnd = false
		return len(node.children) == 0
	}
	ch := rune(word[index])
	child, exists := node.children[ch]
	if !exists {
		return false
	}
	shouldDelete := t.deleteWord(child, word, index+1)
	if shouldDelete {
		delete(node.children, ch)
		return len(node.children) == 0 && !node.isEnd
	}
	return false
}

func (t *Trie) deletePrefix(prefix string) {
	path := []*TrieNode{t.root}
	node := t.root

	for _, ch := range prefix {
		child, exists := node.children[ch]
		if !exists {
			return
		}
		path = append(path, child)
		node = child
	}

	var wordsToDelete []string
	t.collectWords(node, prefix, &wordsToDelete)

	for _, word := range wordsToDelete {
		if entry, exists := t.timers[word]; exists {
			entry.timer.Stop()
			delete(t.timers, word)
		}
	}

	node.children = make(map[rune]*TrieNode)
	node.isEnd = false

	for i := len(path) - 1; i > 0; i-- {
		currentNode := path[i]
		parentNode := path[i-1]
		char := rune(prefix[i-1])

		if len(currentNode.children) == 0 && !currentNode.isEnd {
			delete(parentNode.children, char)
		} else {
			break
		}
	}
}
