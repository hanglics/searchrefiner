package main

import (
	"github.com/hscells/cqr"
	"github.com/hscells/groove"
	"github.com/hscells/groove/combinator"
	"github.com/hscells/groove/stats"
	"log"
	"strconv"
)

func buildAdjTree(query cqr.CommonQueryRepresentation, id, parent, level int, ss *stats.ElasticsearchStatisticsSource) (nid int, t tree) {
	var docs int
	if documents, err := seen.Get(query); err == nil {
		docs = len(documents)
	} else {
		d, err := ss.Execute(groove.NewPipelineQuery("adj", "0", query), ss.SearchOptions())
		if err != nil {
			log.Println("something bad happened")
			log.Fatalln(err)
			panic(err)
		}
		combDocs := make(combinator.Documents, len(d))
		for i, doc := range d {
			id, err := strconv.ParseInt(doc.DocId, 10, 32)
			if err != nil {
				log.Println("something bad happened")
				log.Fatalln(err)
				panic(err)
			}
			combDocs[i] = combinator.Document(id)
		}
		switch q := query.(type) {
		case cqr.Keyword:
			//seen[combinator.HashCQR(query)] = combinator.NewAtom(q, combDocs)
			err := seen.Set(query, combinator.NewAtom(q).Documents(seen))
			if err != nil {
				panic(err)
			}
		}
		docs = len(d)
	}
	switch q := query.(type) {
	case cqr.Keyword:
		t.Nodes = append(t.Nodes, node{id, docs, level, q.StringPretty(), "box"})
		t.Edges = append(t.Edges, edge{parent, id, docs, strconv.Itoa(docs)})
		id++
	case cqr.BooleanQuery:
		t.Nodes = append(t.Nodes, node{id, docs, level, q.StringPretty(), "circle"})
		if parent > 0 {
			t.Edges = append(t.Edges, edge{parent, id, docs, strconv.Itoa(docs)})
		}
		this := id
		id++
		for _, child := range q.Children {
			var nt tree
			id, nt = buildAdjTree(child, id, this, level+1, ss)
			t.Nodes = append(t.Nodes, nt.Nodes...)
			t.Edges = append(t.Edges, nt.Edges...)
		}
	}
	nid += id
	return
}

func buildTreeRec(treeNode combinator.LogicalTreeNode, id, parent, level int, ss *stats.ElasticsearchStatisticsSource) (nid int, t tree) {
	if treeNode == nil {
		log.Printf("treeNode %v was nil (top treeNode?) (id %v) with parent %v at level %v\n", treeNode, id, parent, level)
		return
	}
	log.Printf("combined %v (id %v) with parent %v at level %v\n", treeNode, id, parent, level)
	docs := treeNode.Documents(seen)
	switch n := treeNode.(type) {
	case combinator.Combinator:
		t.Nodes = append(t.Nodes, node{id, len(docs), level, n.String(), "circle"})
		if parent > 0 {
			t.Edges = append(t.Edges, edge{parent, id, len(docs), strconv.Itoa(len(docs))})
		}
		this := id
		id++
		for _, child := range n.Clauses {
			if child == nil {
				log.Printf("child treeNode %v (%v; id: %v) combined with %v and level %v\n", treeNode, child, id, parent, level)
				continue
			}
			var nt tree
			id, nt = buildTreeRec(child, id, this, level+1, ss)
			t.Nodes = append(t.Nodes, nt.Nodes...)
			t.Edges = append(t.Edges, nt.Edges...)
		}
	case combinator.Atom:
		t.Nodes = append(t.Nodes, node{id, len(docs), level, n.String(), "box"})
		t.Edges = append(t.Edges, edge{parent, id, len(docs), strconv.Itoa(len(docs))})
		id++
	case combinator.AdjAtom:
		id, t = buildAdjTree(n.Query(), id, parent, level, ss)
	}
	nid += id
	return
}

func buildTree(node combinator.LogicalTreeNode, ss *stats.ElasticsearchStatisticsSource) (t tree) {
	_, t = buildTreeRec(node, 1, 0, 0, ss)
	log.Println("finished processing query, tree has been constructed")
	return
}