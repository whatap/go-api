package llm

// MaxContentBytes — LogSinkPack content 청킹 단위. 사양 (`llm-logsinkpack-spec.md`) 명시.
const MaxContentBytes = 20000

// SplitContent — content 를 20KB 이하 청크로 분할. UTF-8 multi-byte 경계 보존.
//
// python whatap/counter/tasks/llm_log_sink_task.py::_send_chunked (line 199-217) 알고리즘 동등:
// continuation byte (`b[end] & 0xC0 == 0x80`) 면 lead byte 까지 후퇴.
//
// 빈 입력 → 빈 슬라이스 (caller 가 빈 content 발행 여부 결정).
func SplitContent(content string) []string {
	if content == "" {
		return nil
	}
	b := []byte(content)
	if len(b) <= MaxContentBytes {
		return []string{content}
	}
	var chunks []string
	i := 0
	for i < len(b) {
		end := i + MaxContentBytes
		if end > len(b) {
			end = len(b)
		}
		if end < len(b) {
			// UTF-8 continuation byte (0x80~0xBF) 면 lead byte 까지 후퇴
			for end > i && (b[end]&0xC0) == 0x80 {
				end--
			}
		}
		chunks = append(chunks, string(b[i:end]))
		i = end
	}
	return chunks
}
