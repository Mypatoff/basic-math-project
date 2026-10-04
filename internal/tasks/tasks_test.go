package tasks

import "testing"

func TestGenerateWithDifficultyRanges(t *testing.T) {
	want := map[int]struct{ addSub, mult int }{
		1: {10, 5},
		2: {50, 10},
		3: {100, 12},
	}

	for difficulty, limits := range want {
		for i := 0; i < 500; i++ {
			p, err := GenerateWithDifficulty(difficulty)
			if err != nil {
				t.Fatalf("d=%d: unexpected error: %v", difficulty, err)
			}

			switch p.Op {
			case "+":
				if p.A < 0 || p.A > limits.addSub || p.B < 0 || p.B > limits.addSub {
					t.Fatalf("d=%d +: operand out of range: %+v", difficulty, p)
				}
				if p.Answer != p.A+p.B {
					t.Fatalf("d=%d +: wrong answer: %+v", difficulty, p)
				}
			case "-":
				if p.A < p.B {
					t.Fatalf("d=%d -: would be negative: %+v", difficulty, p)
				}
				if p.Answer < 0 {
					t.Fatalf("d=%d -: negative answer: %+v", difficulty, p)
				}
				if p.Answer != p.A-p.B {
					t.Fatalf("d=%d -: wrong answer: %+v", difficulty, p)
				}
			case "x":
				if p.A < 1 || p.A > limits.mult || p.B < 1 || p.B > limits.mult {
					t.Fatalf("d=%d x: operand out of range: %+v", difficulty, p)
				}
				if p.Answer != p.A*p.B {
					t.Fatalf("d=%d x: wrong answer: %+v", difficulty, p)
				}
			default:
				t.Fatalf("d=%d: unknown operator %q", difficulty, p.Op)
			}
		}
	}
}

func TestGenerateWithDifficultyInvalid(t *testing.T) {
	for _, d := range []int{0, 4, -1, 100} {
		if _, err := GenerateWithDifficulty(d); err == nil {
			t.Errorf("d=%d: expected an error, got nil", d)
		}
	}
}
