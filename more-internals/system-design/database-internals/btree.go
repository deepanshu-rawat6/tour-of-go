package main

import "fmt"

const order = 3 // max children per node

// BTree is a minimal B-tree supporting insert and search.
type BTree struct {
	root *btNode
}

type btNode struct {
	keys     []int
	children []*btNode
	leaf     bool
}

func NewBTree() *BTree {
	return &BTree{root: &btNode{leaf: true}}
}

func (t *BTree) Search(key int) bool {
	return t.root.search(key)
}

func (n *btNode) search(key int) bool {
	i := 0
	for i < len(n.keys) && key > n.keys[i] {
		i++
	}
	if i < len(n.keys) && n.keys[i] == key {
		return true
	}
	if n.leaf {
		return false
	}
	return n.children[i].search(key)
}

func (t *BTree) Insert(key int) {
	r := t.root
	if len(r.keys) == 2*order-1 {
		s := &btNode{children: []*btNode{r}}
		s.splitChild(0)
		t.root = s
	}
	t.root.insertNonFull(key)
}

func (n *btNode) insertNonFull(key int) {
	i := len(n.keys) - 1
	if n.leaf {
		n.keys = append(n.keys, 0)
		for i >= 0 && key < n.keys[i] {
			n.keys[i+1] = n.keys[i]
			i--
		}
		n.keys[i+1] = key
		return
	}
	for i >= 0 && key < n.keys[i] {
		i--
	}
	i++
	if len(n.children[i].keys) == 2*order-1 {
		n.splitChild(i)
		if key > n.keys[i] {
			i++
		}
	}
	n.children[i].insertNonFull(key)
}

func (n *btNode) splitChild(i int) {
	y := n.children[i]
	z := &btNode{leaf: y.leaf}
	mid := order - 1

	// z gets keys after midpoint
	z.keys = append(z.keys, y.keys[mid+1:]...)
	if !y.leaf {
		z.children = append(z.children, y.children[mid+1:]...)
		y.children = y.children[:mid+1]
	}

	// insert median into parent
	n.keys = append(n.keys, 0)
	copy(n.keys[i+1:], n.keys[i:])
	n.keys[i] = y.keys[mid]

	n.children = append(n.children, nil)
	copy(n.children[i+2:], n.children[i+1:])
	n.children[i+1] = z

	y.keys = y.keys[:mid]
}

func (t *BTree) InOrder() []int {
	var result []int
	t.root.inOrder(&result)
	return result
}

func (n *btNode) inOrder(result *[]int) {
	for i, key := range n.keys {
		if !n.leaf {
			n.children[i].inOrder(result)
		}
		*result = append(*result, key)
	}
	if !n.leaf {
		n.children[len(n.keys)].inOrder(result)
	}
}

func demoBTree() {
	fmt.Println("=== B-Tree Demo ===")
	tree := NewBTree()
	for _, v := range []int{10, 20, 5, 6, 12, 30, 7, 17, 3, 25, 50, 1} {
		tree.Insert(v)
	}
	fmt.Println("In-order:", tree.InOrder())
	fmt.Println("Search 12:", tree.Search(12))
	fmt.Println("Search 99:", tree.Search(99))
}
