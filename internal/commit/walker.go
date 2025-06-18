package commit

import (
	"fmt"
	"io"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

type queueEntry struct {
	current *object.Commit
	end     *object.Commit
}

type commitWalker struct {
	queue []*queueEntry
}

// This walker will walk through the commit graph in topo order and newest first
func NewWalker(c *object.Commit) object.CommitIter {
	return &commitWalker{[]*queueEntry{{current: c}}}
}

func (w *commitWalker) Next() (*object.Commit, error) {
	if len(w.queue) == 0 {
		return nil, io.EOF
	}

	entry := w.queue[len(w.queue)-1]
	current := entry.current

	parents := entry.current.Parents()
	var i int
	err := parents.ForEach(func(p *object.Commit) error {
		// reached merge base, remove branch vom queue
		if entry.end != nil && entry.end.Hash == p.Hash {
			w.queue = w.queue[:1]
		}

		// If there are multiple parents, insert parents branch commits by prepending them to the queue
		// Otherwise make the first parent the new current
		if i != 0 {
			mergeBase, err := entry.current.MergeBase(p)
			if err != nil {
				return fmt.Errorf("fetching merge base: %w", err)
			}
			if len(mergeBase) == 0 {
				return fmt.Errorf("could not find merge base of %s and %s", entry.current.Hash, p.Hash)
			}
			w.queue = append(w.queue, &queueEntry{current: p, end: mergeBase[0]})
		} else {
			entry.current = p
		}
		i++
		return nil
	})
	if err != nil {
		return nil, err
	}

	// reached first commit with no parent commit
	if i == 0 {
		w.queue = []*queueEntry{}
	}

	return current, nil
}

func (w *commitWalker) ForEach(cb func(*object.Commit) error) error {
	for {
		c, err := w.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		err = cb(c)
		if err == storer.ErrStop {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *commitWalker) Close() { w.queue = nil }
