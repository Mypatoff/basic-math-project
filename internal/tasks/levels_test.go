package tasks

import "testing"

const levelTestSamples = 300

func TestLevel1AdditionWithin10(t *testing.T) {
	level, _ := LevelByID(1)
	for i := 0; i < levelTestSamples; i++ {
		p := level.Generate()
		if p.Op != "+" {
			t.Fatalf("got op %q, want +", p.Op)
		}
		if p.A < 0 || p.A > 9 || p.B < 0 || p.B > 9 {
			t.Fatalf("operand out of 0-9: %+v", p)
		}
		if p.Answer != p.A+p.B {
			t.Fatalf("bad answer: %+v", p)
		}
	}
}

func TestLevel2AdditionWithin20(t *testing.T) {
	level, _ := LevelByID(2)
	for i := 0; i < levelTestSamples; i++ {
		p := level.Generate()
		if p.Op != "+" {
			t.Fatalf("got op %q, want +", p.Op)
		}
		if p.A < 0 || p.A > 20 || p.B < 0 || p.B > 20 {
			t.Fatalf("operand out of 0-20: %+v", p)
		}
		if p.Answer != p.A+p.B {
			t.Fatalf("bad answer: %+v", p)
		}
	}
}

func TestLevel3SubtractionWithin20(t *testing.T) {
	level, _ := LevelByID(3)
	for i := 0; i < levelTestSamples; i++ {
		p := level.Generate()
		if p.Op != "-" {
			t.Fatalf("got op %q, want -", p.Op)
		}
		if p.A < 0 || p.A > 20 || p.B < 0 || p.B > 20 {
			t.Fatalf("operand out of 0-20: %+v", p)
		}
		if p.A < p.B {
			t.Fatalf("would be negative: %+v", p)
		}
		if p.Answer != p.A-p.B {
			t.Fatalf("bad answer: %+v", p)
		}
	}
}

func TestLevel4AddSubWithin50(t *testing.T) {
	level, _ := LevelByID(4)
	for i := 0; i < levelTestSamples; i++ {
		p := level.Generate()
		if p.A < 0 || p.A > 50 || p.B < 0 || p.B > 50 {
			t.Fatalf("operand out of 0-50: %+v", p)
		}
		switch p.Op {
		case "+":
			if p.Answer != p.A+p.B {
				t.Fatalf("bad answer: %+v", p)
			}
		case "-":
			if p.A < p.B {
				t.Fatalf("would be negative: %+v", p)
			}
			if p.Answer != p.A-p.B {
				t.Fatalf("bad answer: %+v", p)
			}
		default:
			t.Fatalf("unexpected op %q: %+v", p.Op, p)
		}
	}
}

func TestLevel5MultiplicationTables2to5(t *testing.T) {
	level, _ := LevelByID(5)
	for i := 0; i < levelTestSamples; i++ {
		p := level.Generate()
		if p.Op != "x" {
			t.Fatalf("got op %q, want x", p.Op)
		}
		if p.A < 2 || p.A > 5 {
			t.Fatalf("table factor out of 2-5: %+v", p)
		}
		if p.B < 1 || p.B > 10 {
			t.Fatalf("multiplier out of 1-10: %+v", p)
		}
		if p.Answer != p.A*p.B {
			t.Fatalf("bad answer: %+v", p)
		}
	}
}

func TestLevel6MultiplicationTablesUpTo10(t *testing.T) {
	level, _ := LevelByID(6)
	for i := 0; i < levelTestSamples; i++ {
		p := level.Generate()
		if p.Op != "x" {
			t.Fatalf("got op %q, want x", p.Op)
		}
		if p.A < 2 || p.A > 10 {
			t.Fatalf("table factor out of 2-10: %+v", p)
		}
		if p.B < 1 || p.B > 10 {
			t.Fatalf("multiplier out of 1-10: %+v", p)
		}
		if p.Answer != p.A*p.B {
			t.Fatalf("bad answer: %+v", p)
		}
	}
}

func TestLevel7AddSubWithin100(t *testing.T) {
	level, _ := LevelByID(7)
	for i := 0; i < levelTestSamples; i++ {
		p := level.Generate()
		if p.A < 0 || p.A > 100 || p.B < 0 || p.B > 100 {
			t.Fatalf("operand out of 0-100: %+v", p)
		}
		switch p.Op {
		case "+":
			if p.Answer != p.A+p.B {
				t.Fatalf("bad answer: %+v", p)
			}
		case "-":
			if p.A < p.B {
				t.Fatalf("would be negative: %+v", p)
			}
			if p.Answer != p.A-p.B {
				t.Fatalf("bad answer: %+v", p)
			}
		default:
			t.Fatalf("unexpected op %q: %+v", p.Op, p)
		}
	}
}

func TestLevel8ExactDivision(t *testing.T) {
	level, _ := LevelByID(8)
	for i := 0; i < levelTestSamples; i++ {
		p := level.Generate()
		if p.Op != "÷" {
			t.Fatalf("got op %q, want ÷", p.Op)
		}
		if p.B < 2 || p.B > 10 {
			t.Fatalf("divisor out of 2-10: %+v", p)
		}
		if p.A%p.B != 0 {
			t.Fatalf("not an exact division: %+v", p)
		}
		if p.Answer != p.A/p.B {
			t.Fatalf("bad answer: %+v", p)
		}
	}
}

func TestLevel9TwoDigitByOneDigit(t *testing.T) {
	level, _ := LevelByID(9)
	for i := 0; i < levelTestSamples; i++ {
		p := level.Generate()
		if p.Op != "x" {
			t.Fatalf("got op %q, want x", p.Op)
		}
		if p.A < 10 || p.A > 99 {
			t.Fatalf("A not a 2-digit number: %+v", p)
		}
		if p.B < 2 || p.B > 9 {
			t.Fatalf("B not a 1-digit 2-9 number: %+v", p)
		}
		if p.Answer != p.A*p.B {
			t.Fatalf("bad answer: %+v", p)
		}
	}
}

func TestLevel10Mixed(t *testing.T) {
	level, _ := LevelByID(10)
	for i := 0; i < levelTestSamples; i++ {
		p := level.Generate()
		switch p.Op {
		case "+":
			if p.Answer != p.A+p.B {
				t.Fatalf("bad answer: %+v", p)
			}
		case "-":
			if p.A < p.B {
				t.Fatalf("would be negative: %+v", p)
			}
			if p.Answer != p.A-p.B {
				t.Fatalf("bad answer: %+v", p)
			}
		case "x":
			if p.Answer != p.A*p.B {
				t.Fatalf("bad answer: %+v", p)
			}
		case "÷":
			if p.A%p.B != 0 {
				t.Fatalf("not an exact division: %+v", p)
			}
			if p.Answer != p.A/p.B {
				t.Fatalf("bad answer: %+v", p)
			}
		default:
			t.Fatalf("unexpected op %q: %+v", p.Op, p)
		}
	}
}

func TestLevelByIDUnknown(t *testing.T) {
	if _, ok := LevelByID(0); ok {
		t.Error("level 0 should not exist")
	}
	if _, ok := LevelByID(11); ok {
		t.Error("level 11 should not exist")
	}
}
