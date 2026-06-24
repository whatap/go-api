package llm

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSplitContent_Empty(t *testing.T) {
	if got := SplitContent(""); got != nil {
		t.Fatalf("empty: want nil, got %v", got)
	}
}

func TestSplitContent_Short(t *testing.T) {
	got := SplitContent("hello")
	if len(got) != 1 || got[0] != "hello" {
		t.Fatalf("short: want [hello], got %v", got)
	}
}

func TestSplitContent_Exactly20KB(t *testing.T) {
	in := strings.Repeat("a", MaxContentBytes)
	got := SplitContent(in)
	if len(got) != 1 {
		t.Fatalf("exactly 20KB: want 1 chunk, got %d", len(got))
	}
	if len(got[0]) != MaxContentBytes {
		t.Fatalf("exactly 20KB: want %d bytes, got %d", MaxContentBytes, len(got[0]))
	}
}

func TestSplitContent_Over20KB_ASCII(t *testing.T) {
	in := strings.Repeat("a", MaxContentBytes+1)
	got := SplitContent(in)
	if len(got) != 2 {
		t.Fatalf("over 20KB ASCII: want 2 chunks, got %d", len(got))
	}
	if len(got[0]) != MaxContentBytes {
		t.Fatalf("first chunk: want %d, got %d", MaxContentBytes, len(got[0]))
	}
	if len(got[1]) != 1 {
		t.Fatalf("second chunk: want 1, got %d", len(got[1]))
	}
}

func TestSplitContent_MultiByteBoundary(t *testing.T) {
	// 한글 (3-byte UTF-8) 로 정확히 boundary 가까이
	// 6666 한글 (each 3 bytes) = 19998 bytes + 추가 한글 1 = 20001 bytes
	in := strings.Repeat("가", 6667) // 20001 bytes
	got := SplitContent(in)
	if len(got) != 2 {
		t.Fatalf("multi-byte: want 2 chunks, got %d", len(got))
	}
	// 모든 청크가 valid UTF-8 (분할이 multi-byte 시퀀스 중간 안 끊었는지)
	for i, c := range got {
		if !utf8.ValidString(c) {
			t.Fatalf("multi-byte: chunk %d is not valid UTF-8", i)
		}
	}
	// 이어붙이면 원본 복구
	rejoined := got[0] + got[1]
	if rejoined != in {
		t.Fatalf("multi-byte: rejoin mismatch (lengths: chunk0=%d chunk1=%d, total=%d, want=%d)",
			len(got[0]), len(got[1]), len(rejoined), len(in))
	}
}

func TestSplitContent_4ByteEmoji(t *testing.T) {
	// 5000 emoji × 4 bytes = 20000 bytes (정확히)
	in := strings.Repeat("😀", 5000)
	got := SplitContent(in)
	if len(got) != 1 {
		t.Fatalf("4-byte exactly 20000: want 1 chunk, got %d", len(got))
	}

	// 5001 emoji = 20004 bytes — 분할
	in2 := strings.Repeat("😀", 5001)
	got2 := SplitContent(in2)
	if len(got2) != 2 {
		t.Fatalf("4-byte over: want 2 chunks, got %d", len(got2))
	}
	rejoined := got2[0] + got2[1]
	if rejoined != in2 {
		t.Fatalf("4-byte rejoin mismatch")
	}
}

func TestSplitContent_LargeMixed(t *testing.T) {
	// 60001 byte 한글 — 약 3 청크
	in := strings.Repeat("가", 20001) // 20001 × 3 = 60003 bytes
	got := SplitContent(in)
	if len(got) < 3 {
		t.Fatalf("large mixed: want >= 3 chunks, got %d", len(got))
	}
	rejoined := strings.Join(got, "")
	if rejoined != in {
		t.Fatalf("large mixed: rejoin mismatch")
	}
	for i, c := range got[:len(got)-1] {
		if len(c) > MaxContentBytes {
			t.Fatalf("chunk %d exceeds %d bytes: %d", i, MaxContentBytes, len(c))
		}
	}
}
