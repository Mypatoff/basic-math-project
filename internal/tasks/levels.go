package tasks

import "math/rand"

// Level is one practice level: a display title and a function that
// generates one problem at that level's difficulty. Generate always
// returns integer answers; subtraction never goes negative and
// division never leaves a remainder.
type Level struct {
	ID       int
	Title    string
	Generate func() Problem
}

// Levels runs from easiest (1) to hardest (10).
var Levels = []Level{
	{1, "Addition within 10", addRange(0, 9)},
	{2, "Addition within 20", addRange(0, 20)},
	{3, "Subtraction within 20", subRange(0, 20)},
	{4, "Addition & subtraction within 50", addSubRange(0, 50)},
	{5, "Multiplication tables 2-5", mulTable(2, 5)},
	{6, "Multiplication tables up to 10", mulTable(2, 10)},
	{7, "Addition & subtraction within 100", addSubRange(0, 100)},
	{8, "Division (divisor 2-10)", divRange(2, 10, 12)},
	{9, "2-digit times 1-digit", mulDigits()},
	{10, "Mixed practice", mixed()},
}

// LevelByID looks up a level by its ID.
func LevelByID(id int) (Level, bool) {
	for _, level := range Levels {
		if level.ID == id {
			return level, true
		}
	}
	return Level{}, false
}

func addRange(min, max int) func() Problem {
	return func() Problem {
		a, b := min+rand.Intn(max-min+1), min+rand.Intn(max-min+1)
		return Problem{A: a, B: b, Op: "+", Answer: a + b}
	}
}

func subRange(min, max int) func() Problem {
	return func() Problem {
		a, b := min+rand.Intn(max-min+1), min+rand.Intn(max-min+1)
		if b > a {
			a, b = b, a
		}
		return Problem{A: a, B: b, Op: "-", Answer: a - b}
	}
}

func addSubRange(min, max int) func() Problem {
	add, sub := addRange(min, max), subRange(min, max)
	return func() Problem {
		if rand.Intn(2) == 0 {
			return add()
		}
		return sub()
	}
}

// mulTable generates a "times table" problem: A is the table being
// practiced (tableMin-tableMax), B is the multiplier (1-10).
func mulTable(tableMin, tableMax int) func() Problem {
	return func() Problem {
		a := tableMin + rand.Intn(tableMax-tableMin+1)
		b := 1 + rand.Intn(10)
		return Problem{A: a, B: b, Op: "x", Answer: a * b}
	}
}

func mulDigits() func() Problem {
	return func() Problem {
		a := 10 + rand.Intn(90) // 2-digit: 10-99
		b := 2 + rand.Intn(8)   // 1-digit: 2-9
		return Problem{A: a, B: b, Op: "x", Answer: a * b}
	}
}

// divRange picks a divisor and quotient first, then multiplies them
// to get the dividend, so the division is always exact.
func divRange(divisorMin, divisorMax, quotientMax int) func() Problem {
	return func() Problem {
		divisor := divisorMin + rand.Intn(divisorMax-divisorMin+1)
		quotient := 1 + rand.Intn(quotientMax)
		return Problem{A: divisor * quotient, B: divisor, Op: "÷", Answer: quotient}
	}
}

func mixed() func() Problem {
	gens := []func() Problem{
		addSubRange(0, 50),
		mulTable(2, 10),
		divRange(2, 10, 12),
	}
	return func() Problem {
		return gens[rand.Intn(len(gens))]()
	}
}
