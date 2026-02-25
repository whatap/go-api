# Claude Code 기본 규칙

## 프로젝트 관계

**go-api는 apm-go-agent의 서브셋입니다.**

- **apm-go-agent** (`C:\Users\hamma\git\apm-go-agent`): 독립 실행형 다국어 APM 에이전트 (PHP, Python, .NET, Go 지원)
- **go-api** (`C:\Users\hamma\git\whatap-go-sdk\go-api`): Go 애플리케이션 전용 계측 라이브러리 (임베디드 에이전트 포함)

### 주요 차이점

1. **코어 에이전트 로직**: go-api의 `agent/agent/` 디렉토리는 apm-go-agent의 `agent/` 디렉토리에서 복사된 임베디드 버전
2. **Import 경로**:
   - apm-go-agent: `github.com/whatap/apm-go-agent/agent/...`
   - go-api: `github.com/whatap/go-api/agent/agent/...`
3. **코드 유사도**: 핵심 에이전트 로직은 95% 이상 동일 (import 경로 제외)

### 아키텍처

```
apm-go-agent (독립 프로세스)
├── 다국어 지원 (PHP, Python, .NET, Go)
├── Forwarder 모드 (TCP 6600)
└── OTLP 서버

go-api (임베디드 라이브러리)
├── Go 전용
├── 프레임워크 계측 (Gin, Echo, Fiber, GORM 등)
└── 인프로세스 에이전트
```

### 개발 시 주의사항

1. **양방향 동기화**: 핵심 에이전트 로직 변경 시 두 저장소 모두 업데이트 필요
2. **기능 분리**:
   - apm-go-agent 전용: 다국어 지원, Forwarder, OTLP 서버
   - go-api 전용: Go 프레임워크 계측
3. **버전 관리**:
   - apm-go-agent: v1.0.0
   - go-api: v0.5.4 (go-api-inst와 동일 버전)

핵심 에이전트 로직은 import 경로만 다르고 95% 이상 동일합니다.

## 릴리즈 노트 작성 절차

### 1. 버전 규칙
- go-api, go-api-inst 동일 버전 사용
- 배포 버전은 사용자가 지정 (CLAUDE.md 또는 release-guide.md 참조)

### 2. 릴리즈 노트 형식

`go-api/ReleaseNote.md`에 **영문**으로 작성 (apm-go-agent/ReleaseNote.md 표준):

```markdown
## Go agent v0.5.x

Month DD, YYYY

- [New] New feature description.

    Detailed explanation...

    ```ini
    option_name=value
    ```

- [Feature] Existing feature extension.

    Detailed explanation...

- [Fixed] Bug fix description.

    Fix details.

- [Change] Change description.

- [Deprecate] Deprecation notice.

---
```

### 3. 태그 종류 (apm-go-agent 표준)
- `[New]`: 완전히 새로운 기능/역량
- `[Feature]`: 기존 기능 확장/추가
- `[Fixed]`: 버그 수정
- `[Change]`: 기존 동작 변경
- `[Deprecate]`: 사용 중단 예정 기능

### 4. 작성 시 포함 사항
- 날짜 (Month DD, YYYY — 영문)
- 변경 사항 요약 (1줄, 영문)
- 상세 설명 (필요시, 들여쓰기, 영문)
- 설정 옵션 코드 블록 (해당되는 경우)

### 5. 관련 파일
- 내부 원본: `dev-docs/RELEASE_NOTES.md` (go-api 관련 항목 추출하여 작성)
- 형식 참고: `apm-go-agent/ReleaseNote.md`, `apm-go-agent/CLAUDE.md`
- 배포 가이드: `dev-docs/guides/release-guide.md`
