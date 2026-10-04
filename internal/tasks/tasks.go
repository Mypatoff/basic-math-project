// Package tasks generates simple arithmetic problems.
package tasks

import (
	"errors"
	"fmt"
	"math/rand"
)

// Problem is one arithmetic question, e.g. "7 x 8".
type Problem struct {
	A, B   int
	Op     string
	Answer int
}

var ops = []string{"+", "-", "x"}

// Generate returns a random addition, subtraction, or multiplication
// problem. Go has auto-seeded math/rand's global source since Go
// 1.20, so we can call rand.Intn directly without seeding it.
func Generate() Problem {
	op := ops[rand.Intn(len(ops))]
	var a, b, answer int
	switch op {
	case "+":
		a, b = rand.Intn(100), rand.Intn(100)
		answer = a + b
	case "-":
		// Keep subtraction non-negative so results feel natural.
		a, b = rand.Intn(100), rand.Intn(100)
		if b > a {
			a, b = b, a
		}
		answer = a - b
	case "x":
		a, b = rand.Intn(12)+1, rand.Intn(12)+1
		answer = a * b
	}
	return Problem{A: a, B: b, Op: op, Answer: answer}
}

// Question formats the problem for display, e.g. "4 + 9".
func (p Problem) Question() string {
	return fmt.Sprintf("%d %s %d", p.A, p.Op, p.B)
}

// ErrInvalidDifficulty is returned for any difficulty outside 1-3.
var ErrInvalidDifficulty = errors.New("difficulty must be 1, 2, or 3")

// addSubMax and multMax give the (inclusive) number ranges for each
// difficulty: 1 is easiest, 3 is hardest.
func addSubMax(difficulty int) (addSub, mult int, err error) {
	switch difficulty {
	case 1:
		return 10, 5, nil
	case 2:
		return 50, 10, nil
	case 3:
		return 100, 12, nil
	default:
		return 0, 0, ErrInvalidDifficulty
	}
}

// GenerateWithDifficulty is like Generate but scales the operand
// ranges to a difficulty level.
func GenerateWithDifficulty(difficulty int) (Problem, error) {
	addSub, mult, err := addSubMax(difficulty)
	if err != nil {
		return Problem{}, err
	}

	op := ops[rand.Intn(len(ops))]
	var a, b, answer int
	switch op {
	case "+":
		a, b = rand.Intn(addSub+1), rand.Intn(addSub+1)
		answer = a + b
	case "-":
		a, b = rand.Intn(addSub+1), rand.Intn(addSub+1)
		if b > a {
			a, b = b, a
		}
		answer = a - b
	case "x":
		a, b = rand.Intn(mult)+1, rand.Intn(mult)+1
		answer = a * b
	}
	return Problem{A: a, B: b, Op: op, Answer: answer}, nil
}
