package sqlutil

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// --- 기본 기능 테스트 ---

func TestProcess_BasicSelect(t *testing.T) {
	sql := "SELECT * FROM users WHERE name = 'john' AND age = 25"
	els := NewEscapeLiteralSQL(sql)
	els.Process()

	parsed := els.GetParsedSql()
	param := els.GetParameter()

	// 문자열 리터럴 'john'이 '#'으로 치환되어야 함
	if strings.Contains(parsed, "john") {
		t.Errorf("parsed SQL should not contain literal 'john': %s", parsed)
	}
	if !strings.Contains(parsed, "'#'") {
		t.Errorf("parsed SQL should contain substitute '#': %s", parsed)
	}

	// 숫자 리터럴 25도 치환되어야 함
	if strings.Contains(parsed, "25") {
		t.Errorf("parsed SQL should not contain literal '25': %s", parsed)
	}

	// 파라미터에 추출된 값 포함
	if !strings.Contains(param, "john") {
		t.Errorf("param should contain 'john': %s", param)
	}
	if !strings.Contains(param, "25") {
		t.Errorf("param should contain '25': %s", param)
	}

	// SQL 타입 = 'S' (SELECT)
	if els.SqlType != 'S' {
		t.Errorf("SqlType should be 'S', got '%c'", els.SqlType)
	}
}

func TestProcess_Insert(t *testing.T) {
	sql := "INSERT INTO users (name, age) VALUES ('alice', 30)"
	els := NewEscapeLiteralSQL(sql)
	els.Process()

	parsed := els.GetParsedSql()
	if strings.Contains(parsed, "alice") {
		t.Errorf("parsed SQL should not contain literal 'alice': %s", parsed)
	}
	if els.SqlType != 'I' {
		t.Errorf("SqlType should be 'I', got '%c'", els.SqlType)
	}
}

func TestProcess_Update(t *testing.T) {
	sql := "UPDATE users SET name = 'bob' WHERE id = 1"
	els := NewEscapeLiteralSQL(sql)
	els.Process()

	if els.SqlType != 'U' {
		t.Errorf("SqlType should be 'U', got '%c'", els.SqlType)
	}
}

func TestProcess_Delete(t *testing.T) {
	sql := "DELETE FROM users WHERE id = 99"
	els := NewEscapeLiteralSQL(sql)
	els.Process()

	parsed := els.GetParsedSql()
	if strings.Contains(parsed, "99") {
		t.Errorf("parsed SQL should not contain literal '99': %s", parsed)
	}
	if els.SqlType != 'D' {
		t.Errorf("SqlType should be 'D', got '%c'", els.SqlType)
	}
}

func TestProcess_CommentIncluded(t *testing.T) {
	sql := "SELECT /* hint */ * FROM users WHERE id = 1"
	els := NewEscapeLiteralSQL(sql)
	els.SetIncludeComment(true)
	els.Process()

	parsed := els.GetParsedSql()
	if !strings.Contains(parsed, "/* hint */") {
		t.Errorf("parsed SQL should include comment: %s", parsed)
	}
}

func TestProcess_CommentExcluded(t *testing.T) {
	sql := "SELECT /* hint */ * FROM users WHERE id = 1"
	els := NewEscapeLiteralSQL(sql)
	els.SetIncludeComment(false)
	els.Process()

	parsed := els.GetParsedSql()
	if strings.Contains(parsed, "hint") {
		t.Errorf("parsed SQL should not include comment when excluded: %s", parsed)
	}
}

func TestProcess_EscapedQuote(t *testing.T) {
	// SQL에서 따옴표 두 개 연속 = 이스케이프된 따옴표
	sql := "SELECT * FROM users WHERE name = 'it''s'"
	els := NewEscapeLiteralSQL(sql)
	els.Process()

	param := els.GetParameter()
	// 파라미터에 이스케이프된 따옴표가 포함되어야 함
	if !strings.Contains(param, "it") {
		t.Errorf("param should contain escaped quote content: %s", param)
	}
}

func TestProcess_MultipleStringLiterals(t *testing.T) {
	sql := "SELECT * FROM users WHERE name = 'john' AND city = 'seoul'"
	els := NewEscapeLiteralSQL(sql)
	els.Process()

	param := els.GetParameter()
	if !strings.Contains(param, "john") {
		t.Errorf("param should contain 'john': %s", param)
	}
	if !strings.Contains(param, "seoul") {
		t.Errorf("param should contain 'seoul': %s", param)
	}
}

func TestProcess_NoLiterals(t *testing.T) {
	sql := "SELECT count(*) FROM users"
	els := NewEscapeLiteralSQL(sql)
	els.Process()

	parsed := els.GetParsedSql()
	param := els.GetParameter()

	if parsed != sql {
		t.Errorf("parsed SQL should be unchanged: got %q, want %q", parsed, sql)
	}
	if param != "" {
		t.Errorf("param should be empty: %s", param)
	}
}

func TestProcess_LineComment(t *testing.T) {
	sql := "SELECT * FROM users -- where clause\nWHERE id = 1"
	els := NewEscapeLiteralSQL(sql)
	els.SetIncludeComment(true)
	els.Process()

	parsed := els.GetParsedSql()
	// WHERE는 주석 뒤 줄에 있으므로 포함되어야 함
	if !strings.Contains(parsed, "WHERE") {
		t.Errorf("parsed SQL should contain WHERE after line comment: %s", parsed)
	}
}

// --- §151: 대형 SQL 성능 테스트 ---

// generateLargeSQL은 지정된 크기의 SQL을 생성한다.
// 문자열 리터럴을 많이 포함하여 _qutation()의 param.String() 호출을 유발한다.
func generateLargeSQL(targetSize int) string {
	var b strings.Builder
	b.WriteString("INSERT INTO data (col) VALUES ")
	for b.Len() < targetSize {
		if b.Len() > len("INSERT INTO data (col) VALUES ") {
			b.WriteString(", ")
		}
		b.WriteString("('")
		// 100자 문자열 리터럴
		for i := 0; i < 100; i++ {
			b.WriteByte('a' + byte(i%26))
		}
		b.WriteString("')")
	}
	return b.String()
}

// generateLargeSQLNumbers는 숫자 리터럴이 많은 대형 SQL을 생성한다.
// _number()의 param.String() 호출을 유발한다.
func generateLargeSQLNumbers(targetSize int) string {
	var b strings.Builder
	b.WriteString("INSERT INTO data (col) VALUES ")
	i := 0
	for b.Len() < targetSize {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("(")
		b.WriteString(fmt.Sprintf("%d", i))
		b.WriteString(")")
		i++
	}
	return b.String()
}

func TestProcess_LargeSQL_Completes(t *testing.T) {
	// 1MB SQL 생성
	sql := generateLargeSQL(1024 * 1024)
	t.Logf("SQL length: %d bytes (%.1f MB)", len(sql), float64(len(sql))/(1024*1024))

	start := time.Now()
	els := NewEscapeLiteralSQL(sql)
	els.Process()
	elapsed := time.Since(start)

	t.Logf("Process time: %v", elapsed)

	// 10초 이내에 완료되어야 함 (O(n²)이면 훨씬 오래 걸림)
	if elapsed > 10*time.Second {
		t.Errorf("Process took too long for 1MB SQL: %v (expected < 10s)", elapsed)
	}
}

func TestProcess_32KB_SQL(t *testing.T) {
	// maxSqlNormalizeLength (32KB) 경계값 테스트
	sql := generateLargeSQL(32 * 1024)
	t.Logf("SQL length: %d bytes", len(sql))

	start := time.Now()
	els := NewEscapeLiteralSQL(sql)
	els.Process()
	elapsed := time.Since(start)

	t.Logf("Process time: %v", elapsed)

	if elapsed > 5*time.Second {
		t.Errorf("Process took too long for 32KB SQL: %v", elapsed)
	}
}

func TestProcess_LargeSQL_Numbers(t *testing.T) {
	// 숫자 리터럴이 많은 대형 SQL
	sql := generateLargeSQLNumbers(512 * 1024)
	t.Logf("SQL length: %d bytes (%.1f KB)", len(sql), float64(len(sql))/1024)

	start := time.Now()
	els := NewEscapeLiteralSQL(sql)
	els.Process()
	elapsed := time.Since(start)

	t.Logf("Process time: %v", elapsed)

	if elapsed > 10*time.Second {
		t.Errorf("Process took too long for large numeric SQL: %v", elapsed)
	}
}

// --- 벤치마크 ---

func BenchmarkProcess_SmallSQL(b *testing.B) {
	sql := "SELECT * FROM users WHERE name = 'john' AND age = 25 AND city = 'seoul'"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		els := NewEscapeLiteralSQL(sql)
		els.Process()
	}
}

func BenchmarkProcess_1KB(b *testing.B) {
	sql := generateLargeSQL(1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		els := NewEscapeLiteralSQL(sql)
		els.Process()
	}
}

func BenchmarkProcess_32KB(b *testing.B) {
	sql := generateLargeSQL(32 * 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		els := NewEscapeLiteralSQL(sql)
		els.Process()
	}
}

func BenchmarkProcess_128KB(b *testing.B) {
	sql := generateLargeSQL(128 * 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		els := NewEscapeLiteralSQL(sql)
		els.Process()
	}
}

func BenchmarkProcess_512KB(b *testing.B) {
	sql := generateLargeSQL(512 * 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		els := NewEscapeLiteralSQL(sql)
		els.Process()
	}
}

func BenchmarkProcess_1MB(b *testing.B) {
	sql := generateLargeSQL(1024 * 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		els := NewEscapeLiteralSQL(sql)
		els.Process()
	}
}

func BenchmarkProcess_1MB_Numbers(b *testing.B) {
	sql := generateLargeSQLNumbers(1024 * 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		els := NewEscapeLiteralSQL(sql)
		els.Process()
	}
}
