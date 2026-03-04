# dabom-simulator-usage

DABOM 플랫폼의 데이터 사용량 이벤트를 시뮬레이션하여 Kafka로 발행하는 부하 생성기입니다.

가족 그룹(Family) 단위로 구성원(Customer)의 `DATA_USAGE` 이벤트를 생성하며, TPS 제어, 다양한 부하 패턴, HTTP API를 통한 실시간 조정을 지원합니다.

## 요구사항

- Go 1.24+
- Kafka 브로커 (Apache Kafka 3.x)
- Docker / Docker Compose (컨테이너 실행 시)

## 빌드

```bash
go build -o simulator-usage ./cmd/simulator/
```

## 실행

### 로컬 실행 (dev)

Kafka가 `localhost:9092`에 실행 중이어야 합니다.

```bash
go run ./cmd/simulator -config configs/config.dev.yaml
```

### Docker Compose (Kafka 포함)

Kafka와 시뮬레이터를 함께 띄웁니다.

```bash
docker compose up -d
```

### Docker 단독 빌드/실행

이미 외부에 Kafka가 있는 경우:

```bash
docker build -t dabom-simulator-usage .
docker run --rm \
  -e KAFKA_BROKERS=host.docker.internal:9092 \
  -p 8080:8080 -p 9090:9090 \
  dabom-simulator-usage
```

## 설정

설정 파일은 `configs/` 디렉토리에 있습니다.

| 파일 | 용도 |
|---|---|
| `config.dev.yaml` | 로컬 개발/테스트 (TPS 100, 가족 1,000개, debug 로그) |
| `config.yaml` | 기본/운영 (TPS 5,000, 가족 250,000개) |
| `config.prod.yaml` | 프로덕션 |

### 주요 설정 항목

```yaml
kafka:
  brokers: ["localhost:9092"]
  topic: "usage-events"

simulation:
  mode: "constant"        # constant | ramp-up | burst | realistic
  tps: 100                # 초당 발행 이벤트 수
  workerCount: 4          # 0이면 CPU 코어 수 * 2

  families:
    count: 1000           # 시뮬레이션할 가족 수
    maxMembers: 6         # 가족당 최대 구성원 수

  # 고정 타겟 설정 시 해당 ID들로만 이벤트 생성 (랜덤 대체)
  fixedTargets:
    - familyId: 12345
      customerIds: [100001, 100002, 100003]
    - familyId: 67890
      customerIds: [200001, 200002]

server:
  controlPort: 8080       # HTTP 제어 API
  metricsPort: 9090       # Prometheus 메트릭
```

## Kafka 메시지 검증

### 1. 토픽 확인

Kafka가 Docker 컨테이너(`dabom-kafka`)로 실행 중인 경우:

```bash
# 토픽 목록
docker exec dabom-kafka /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 --list

# 토픽 상세 정보
docker exec dabom-kafka /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 --describe --topic usage-events
```

### 2. 콘솔 컨슈머로 메시지 확인

터미널을 하나 열고 컨슈머를 대기시킵니다:

```bash
docker exec -it dabom-kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 --topic usage-events --from-beginning
```

### 3. 시뮬레이터 실행

다른 터미널에서 시뮬레이터를 실행합니다:

```bash
go run ./cmd/simulator -config configs/config.dev.yaml
```

### 4. 메시지 형식 확인

컨슈머에 다음과 같은 JSON 메시지가 출력되면 정상입니다:

```json
{
  "eventId": "a1b2c3d4-...",
  "eventType": "DATA_USAGE",
  "timestamp": "2025-01-15T14:30:22.123",
  "payload": {
    "eventId": "evt_e5f6g7h8-...",
    "familyId": 100042,
    "customerId": 200187,
    "appId": "youtube",
    "bytesUsed": 5242880,
    "metadata": {
      "deviceId": "dev-abcdef12",
      "networkType": "5G"
    }
  }
}
```

## HTTP 제어 API

시뮬레이터 실행 중 `localhost:8080`으로 실시간 제어가 가능합니다.

### 상태 확인

```bash
# 헬스 체크
curl http://localhost:8080/health

# 시뮬레이터 상태 (발행 수, TPS, 가동 시간 등)
curl http://localhost:8080/status
```

### 시작/정지

```bash
curl -X POST http://localhost:8080/control/start
curl -X POST http://localhost:8080/control/stop
```

### TPS 변경

```bash
# TPS를 10으로 낮춰서 메시지를 천천히 확인
curl -X PUT http://localhost:8080/config/tps \
  -H "Content-Type: application/json" \
  -d '{"tps": 10}'
```

### 부하 패턴 변경

```bash
# constant: 고정 TPS
curl -X PUT http://localhost:8080/config/mode \
  -H "Content-Type: application/json" \
  -d '{"mode": "constant"}'

# ramp-up: 점진적 증가
curl -X PUT http://localhost:8080/config/mode \
  -H "Content-Type: application/json" \
  -d '{"mode": "ramp-up", "startTps": 10, "targetTps": 500, "durationSeconds": 60}'

# burst: 주기적 버스트
curl -X PUT http://localhost:8080/config/mode \
  -H "Content-Type: application/json" \
  -d '{"mode": "burst"}'

# realistic: 시간대별 변동 (새벽 0.2x ~ 저녁 1.8x)
curl -X PUT http://localhost:8080/config/mode \
  -H "Content-Type: application/json" \
  -d '{"mode": "realistic"}'
```

### 수동 버스트

```bash
# 5초간 1000건 발행
curl -X POST http://localhost:8080/control/burst \
  -H "Content-Type: application/json" \
  -d '{"count": 1000, "durationSeconds": 5}'
```

### 버스트 설정 변경

```bash
curl -X PUT http://localhost:8080/config/burst \
  -H "Content-Type: application/json" \
  -d '{"baseTps": 100, "burstTps": 500, "burstDurationSeconds": 5, "intervalSeconds": 30}'
```

### 고정 타겟 (Fixed Targets)

특정 `familyId`/`customerId`로만 이벤트를 생성합니다. 설정하면 랜덤 시뮬레이션을 대체하고, 해제하면 랜덤 모드로 복귀합니다.

```bash
# 고정 타겟 설정 — 지정한 ID들로만 이벤트 생성
curl -X PUT http://localhost:8080/config/fixed-targets \
  -H "Content-Type: application/json" \
  -d '{"targets": [{"familyId": 12345, "customerIds": [100001, 100002]}, {"familyId": 67890, "customerIds": [200001]}]}'

# 고정 타겟 해제 — 랜덤 모드로 복귀
curl -X DELETE http://localhost:8080/config/fixed-targets
```

설정 파일(`config.dev.yaml`)에서도 지정할 수 있습니다:

```yaml
simulation:
  fixedTargets:
    - familyId: 12345
      customerIds: [100001, 100002, 100003]
    - familyId: 67890
      customerIds: [200001, 200002]
```

## Prometheus 메트릭

`localhost:9090/metrics`에서 Prometheus 형식으로 수집 가능합니다.

```bash
curl http://localhost:9090/metrics
```

주요 메트릭:

| 메트릭 | 타입 | 설명 |
|---|---|---|
| `simulator_events_published_total` | Counter | 발행 성공 이벤트 수 |
| `simulator_events_failed_total` | Counter | 발행 실패 이벤트 수 |
| `simulator_events_published_bytes_total` | Counter | 발행된 바이트 수 |
| `simulator_current_tps` | Gauge | 설정된 TPS |
| `simulator_actual_tps` | Gauge | 실측 TPS |
| `simulator_publish_latency_seconds` | Histogram | Kafka 발행 지연 시간 |
| `simulator_worker_active` | Gauge | 활성 워커 수 |
| `simulator_uptime_seconds` | Gauge | 가동 시간 |

## 테스트

```bash
go test ./...
```

## 프로젝트 구조

```
cmd/simulator/main.go          # 엔트리포인트
configs/
  config.yaml                   # 기본 설정
  config.dev.yaml               # 개발용 설정
  config.prod.yaml              # 운영용 설정
internal/
  api/                          # HTTP 제어 API, Simulator 오케스트레이션
  config/                       # 설정 로드
  generator/                    # 가족/이벤트 생성
  metrics/                      # Prometheus 메트릭
  producer/                     # Kafka 프로듀서, 워커 풀
  ratelimit/                    # TPS 제어, 부하 패턴 (constant/ramp-up/burst/realistic)
Dockerfile                      # 멀티스테이지 빌드
docker-compose.yaml             # Kafka + 시뮬레이터 통합 실행
```
