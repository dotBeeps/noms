package shared

import "testing"

func TestJoinWithGutterEqualLines(t *testing.T) {
	left := "A\nB"
	right := "1\n2"
	sep := " | "

	got := JoinWithGutter(left, right, sep, 4)
	want := JoinHorizontalRaw(left, right, sep)

	if got != want {
		t.Fatalf("JoinWithGutter() = %q, want %q", got, want)
	}
}

func TestJoinWithGutterLeftShorter(t *testing.T) {
	left := "L1\nL2"
	right := "R1\nR2\nR3\nR4\nR5"

	got := JoinWithGutter(left, right, "  ", 4)
	want := "L1  R1\nL2  R2\n      R3\n      R4\n      R5"

	if got != want {
		t.Fatalf("JoinWithGutter() = %q, want %q", got, want)
	}
}

func TestJoinWithGutterRightShorter(t *testing.T) {
	left := "L1\nL2\nL3\nL4\nL5"
	right := "R1\nR2"

	got := JoinWithGutter(left, right, "  ", 4)
	want := "L1  R1\nL2  R2\nL3\nL4\nL5"

	if got != want {
		t.Fatalf("JoinWithGutter() = %q, want %q", got, want)
	}
}

func TestJoinWithGutterEmptyLeft(t *testing.T) {
	right := "R1\nR2\nR3"

	got := JoinWithGutter("", right, "  ", 4)
	want := "      R1\n      R2\n      R3"

	if got != want {
		t.Fatalf("JoinWithGutter() = %q, want %q", got, want)
	}
}
