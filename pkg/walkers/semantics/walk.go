// Copyright 2019 Yaacov Zamir <kobi.zamir@gmail.com>
// and other contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Author: 2019 Nimrod Shneor <nimrodshn@gmail.com>
// Author: 2019 Yaacov Zamir <kobi.zamir@gmail.com>

// Package semantics implements TSL tree semantics.
package semantics

import (
	"fmt"
	"regexp"

	"github.com/yaacov/tsl/pkg/tsl"
)

// Doc represent one document in our in-memmory data base.
type Doc map[string]interface{}

// Walk travel the TSL tree and implements search semantics.
//
// Users can call the Walk method to check if a document compiles to `true` or `false`
// when applied to a tsl tree.
//
// Example:
//  	record :=  map[string]string{
//  		"title":       "A good book",
//  		"author":      "Joe",
//  		"spec.pages":  14,
//  		"spec.rating": 5,
//  	}
//
//  	// Check if our record complie with our tsl tree.
//  	//
//  	// For example:
//  	//   if our tsl tree represents the tsl phrase "author = 'Joe'"
//  	//   we will get the boolean value `true` for our record.
//  	//
//  	//   if our tsl tree represents the tsl phrase "spec.pages > 50"
//  	//   we will get the boolean value `false` for our record.
//  	compliance, err = semantics.Walk(tree, record)
//
func Walk(n tsl.Node, book Doc) (bool, error) {
	l := n.Left.(tsl.Node)

	// Check for identifiers.
	if l.Func == tsl.IdentOp {
		newNode, err := handleIdent(n, book)
		if err != nil {
			return false, err
		}
		return Walk(newNode, book)
	}

	// Implement tree semantics.
	switch n.Func {
	case tsl.EqOp, tsl.NotEqOp, tsl.LtOp, tsl.LteOp, tsl.GtOp, tsl.GteOp, tsl.RegexOp, tsl.NotRegexOp,
		tsl.BetweenOp, tsl.NotBetweenOp, tsl.NotInOp, tsl.InOp:
		r := n.Right.(tsl.Node)

		switch l.Func {
		case tsl.StringOp:
			if r.Func == tsl.StringOp {
				return handleStringOp(n, book)
			}
			if r.Func == tsl.ArrayOp {
				return handleStringArrayOp(n, book)
			}
		case tsl.NumberOp:
			if r.Func == tsl.NumberOp {
				return handleNumberOp(n, book)
			}
			if r.Func == tsl.ArrayOp {
				return handleNumberArrayOp(n, book)
			}
		case tsl.NullOp:
			// Any comparison operation on a null element is false.
			return false, nil
		}
	case tsl.IsNotNilOp:
		return l.Func != tsl.NullOp, nil
	case tsl.IsNilOp:
		return l.Func == tsl.NullOp, nil
	case tsl.AndOp, tsl.OrOp:
		return handleLogicalOp(n, book)
	}

	return false, tsl.UnexpectedLiteralError{Literal: n.Func}
}

func handleIdent(n tsl.Node, book Doc) (tsl.Node, error) {
	l := n.Left.(tsl.Node)

	switch v := book[l.Left.(string)].(type) {
	case string:
		n.Left = tsl.Node{
			Func: tsl.StringOp,
			Left: v,
		}
	case nil:
		n.Left = tsl.Node{
			Func: tsl.NullOp,
			Left: nil,
		}
	case bool:
		val := "false"
		if v {
			val = "true"
		}
		n.Left = tsl.Node{
			Func: tsl.StringOp,
			Left: val,
		}
	case float32:
		n.Left = tsl.Node{
			Func: tsl.NumberOp,
			Left: float64(v),
		}
	case float64:
		n.Left = tsl.Node{
			Func: tsl.NumberOp,
			Left: v,
		}
	case int32:
		n.Left = tsl.Node{
			Func: tsl.NumberOp,
			Left: float64(v),
		}
	case int64:
		n.Left = tsl.Node{
			Func: tsl.NumberOp,
			Left: float64(v),
		}
	case uint32:
		n.Left = tsl.Node{
			Func: tsl.NumberOp,
			Left: float64(v),
		}
	case uint64:
		n.Left = tsl.Node{
			Func: tsl.NumberOp,
			Left: float64(v),
		}
	case int:
		n.Left = tsl.Node{
			Func: tsl.NumberOp,
			Left: float64(v),
		}
	case uint:
		n.Left = tsl.Node{
			Func: tsl.NumberOp,
			Left: float64(v),
		}
	default:
		return n, tsl.UnexpectedLiteralError{Literal: fmt.Sprintf("%s[%v]", l.Left.(string), v)}
	}

	return n, nil
}

func handleStringOp(n tsl.Node, book Doc) (bool, error) {
	l := n.Left.(tsl.Node)
	r := n.Right.(tsl.Node)

	left := l.Left.(string)
	right := r.Left.(string)

	switch n.Func {
	case tsl.EqOp:
		return left == right, nil
	case tsl.NotEqOp:
		return left != right, nil
	case tsl.LtOp:
		return left < right, nil
	case tsl.LteOp:
		return left <= right, nil
	case tsl.GtOp:
		return left > right, nil
	case tsl.GteOp:
		return left >= right, nil
	case tsl.RegexOp:
		valid, err := regexp.Compile(right)
		if err != nil {
			return false, tsl.UnexpectedLiteralError{Literal: right}
		}
		return valid.MatchString(left), nil
	case tsl.NotRegexOp:
		valid, err := regexp.Compile(right)
		if err != nil {
			return false, tsl.UnexpectedLiteralError{Literal: right}
		}
		return !valid.MatchString(left), nil
	}

	return false, tsl.UnexpectedLiteralError{Literal: n.Func}
}

func handleNumberOp(n tsl.Node, book Doc) (bool, error) {
	l := n.Left.(tsl.Node)
	r := n.Right.(tsl.Node)

	left := l.Left.(float64)
	right := r.Left.(float64)

	switch n.Func {
	case tsl.EqOp:
		return left == right, nil
	case tsl.NotEqOp:
		return left != right, nil
	case tsl.LtOp:
		return left < right, nil
	case tsl.LteOp:
		return left <= right, nil
	case tsl.GtOp:
		return left > right, nil
	case tsl.GteOp:
		return left >= right, nil
	}

	return false, tsl.UnexpectedLiteralError{Literal: n.Func}
}

func handleStringArrayOp(n tsl.Node, book Doc) (bool, error) {
	l := n.Left.(tsl.Node)
	r := n.Right.(tsl.Node)

	left := l.Left.(string)
	right := r.Right.([]tsl.Node)

	switch n.Func {
	case tsl.BetweenOp:
		begin := right[0].Left.(string)
		end := right[1].Left.(string)
		return left >= begin && left < end, nil
	case tsl.NotBetweenOp:
		begin := right[0].Left.(string)
		end := right[1].Left.(string)
		return left < begin || left >= end, nil
	case tsl.InOp:
		b := false
		for _, node := range right {
			b = b || left == node.Left.(string)
		}
		return b, nil
	case tsl.NotInOp:
		b := true
		for _, node := range right {
			b = b && left != node.Left.(string)
		}
		return b, nil
	}

	return false, tsl.UnexpectedLiteralError{Literal: n.Func}
}

func handleNumberArrayOp(n tsl.Node, book Doc) (bool, error) {
	l := n.Left.(tsl.Node)
	r := n.Right.(tsl.Node)

	left := l.Left.(float64)
	right := r.Right.([]tsl.Node)

	switch n.Func {
	case tsl.BetweenOp:
		begin := right[0].Left.(float64)
		end := right[1].Left.(float64)
		return left >= begin && left < end, nil
	case tsl.NotBetweenOp:
		begin := right[0].Left.(float64)
		end := right[1].Left.(float64)
		return left < begin || left >= end, nil
	case tsl.InOp:
		b := false
		for _, node := range right {
			b = b || left == node.Left.(float64)
		}
		return b, nil
	case tsl.NotInOp:
		b := true
		for _, node := range right {
			b = b && left != node.Left.(float64)
		}
		return b, nil
	}

	return false, tsl.UnexpectedLiteralError{Literal: n.Func}
}

func handleLogicalOp(n tsl.Node, book Doc) (bool, error) {
	l := n.Left.(tsl.Node)
	r := n.Right.(tsl.Node)

	right, err := Walk(r, book)
	if err != nil {
		return false, err
	}
	left, err := Walk(l, book)
	if err != nil {
		return false, err
	}

	switch n.Func {
	case tsl.AndOp:
		return right && left, nil
	case tsl.OrOp:
		return right || left, nil
	}

	return false, tsl.UnexpectedLiteralError{Literal: n.Func}
}
