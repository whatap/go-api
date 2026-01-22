# go-api 릴리스 체크리스트

## 릴리스 전 필수 점검

### 1. 코드 품질 점검

#### 불필요한 로그 정리
- [ ] `fmt.Print` 계열 제거 또는 logutil 사용
- [ ] `log.Print` 계열 conf.Debug 감싸기 확인
- [ ] 테스트/디버그용 코드 제거

```bash
# fmt.Print 검색
grep -rn "fmt\.Print" --include="*.go" | grep -v "_test\.go" | grep -v "//.*fmt"

# log.Print 검색 (conf.Debug 없는 것)
grep -rn "log\.Print" --include="*.go" | grep -v "_test\.go" | grep -v "//.*log"
```

#### reflect 사용 점검
- [ ] `reflect` 패키지 사용 최소화 (성능 영향)
- [ ] 불필요한 reflect import 제거

```bash
# reflect 사용 검색
grep -rn "reflect\." --include="*.go" | grep -v "_test\.go"
```

#### Dead code 정리
- [ ] 호출되지 않는 함수 제거
- [ ] 주석 처리된 코드 정리

### 2. 테스트

- [ ] 단위 테스트 통과
- [ ] Docker 테스트 통과 (testapps)
- [ ] 분산 추적 (mtid) 테스트

```bash
# 단위 테스트
go test ./...

# Docker 테스트 (WSL)
cd testapps && docker-compose up --build
```

### 3. 문서 업데이트

- [ ] CHANGELOG.md 업데이트
- [ ] README.md 버전 정보 업데이트
- [ ] API 변경사항 문서화

### 4. 버전 태깅

```bash
# 버전 태그
git tag -a v1.x.x -m "Release v1.x.x"
git push origin v1.x.x
```

---

## 체크리스트 이력

| 날짜 | 점검 항목 | 상태 |
|------|----------|------|
| 2026-01-15 | 불필요한 로그 정리 | 완료 (gid/unsafe.go, Config.go) |
| 2026-01-15 | reflect 사용 점검 | 완료 (whatapsarama type switch 개선) |

---

## 허용된 예외

### 의도적인 로그 출력
| 파일 | 이유 |
|------|------|
| `agent/util/logutil/Logger.go` | 내부 로깅 유틸리티 |
| `instrumentation/fmt/whatapfmt/fmt.go` | 사용자 코드 래핑 |

### Dead code (정리 대상)
| 파일 | 함수 | 비고 |
|------|------|------|
| `agent/agent/topology/StatusDetector.go` | `localAddresses()` | 호출 안 됨 |
| `agent/agent/trace/ServiceURLPatternDetector.go` | `ServiceUTLPatternDetectorMain()` | 테스트용 |
| `whatap.go` | `main()` | 테스트용 (라이브러리) |
