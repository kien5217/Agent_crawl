# Agent Crawl

Agent Crawl là hệ thống thu thập và xử lý nội dung theo pipeline Discovery -> Crawl/Extract -> Classification -> Learning, hiện đã tách thành hai phần:
- backend (Go): domain logic, pipeline, API, persistence.
- frontend (React + Vite): dashboard vận hành và quan sát workflow.

## 1. Bức tranh kiến trúc tổng thể

### 1.1 Context architecture

```text
+---------------------------+         HTTP/JSON         +---------------------------+
| Frontend (React + Vite)  | <-----------------------> | Backend API (Go net/http) |
| - Documents UI           |                           | - /api/topics             |
| - Workflow UI            |                           | - /api/documents          |
| - Trigger schedule       |                           | - /api/schedule           |
+---------------------------+                           +-------------+-------------+
                                                                      |
                                                                      | Repository interfaces
                                                                      v
                                                          +-----------+------------+
                                                          | Infrastructure (Go)    |
                                                          | - Discovery (RSS/Site) |
                                                          | - Fetcher/Extractor    |
                                                          | - Classifier           |
                                                          | - ML pipeline          |
                                                          | - Postgres Store       |
                                                          +-----------+------------+
                                                                      |
                                                                      v
                                                          +------------------------+
                                                          | PostgreSQL             |
                                                          | queue, documents, ML,  |
                                                          | workflow executions     |
                                                          +------------------------+
```

### 1.2 Layered architecture (backend)

```text
cmd/main.go
  -> internal/application
     -> api           (HTTP handlers + routing)
     -> cli           (batch/ops commands)
     -> schedule      (discovery orchestration)
     -> worker        (crawl processing)
     -> learning      (weak_label, train, select)
     -> orchestrator  (workflow + step execution)
     -> loader        (YAML config loader)

  -> internal/domain
     -> model         (entities)
     -> repository    (interfaces/contracts)
     -> config        (AppConfig)

  -> internal/infrastructure
     -> discovery, fetcher, extract, classify, machine_learning
     -> persistence/postgres (repository implementations)
```

### 1.3 Current workspace structure

```text
Agent_Crawl/
  backend/
    cmd/
    config/
    internal/
    migrations/
    go.mod
    go.sum
  frontend/
    src/
    package.json
    vite.config.ts
  README.md
```

### 1.4 Runtime topology

- Backend service chạy bằng lệnh api command, expose HTTP API và phục vụ static frontend build từ ../frontend/dist.
- Frontend dev mode chạy Vite tại localhost:5173, proxy /api về localhost:8080.
- Data plane tập trung ở PostgreSQL.

## 2. Technical design chi tiết

### 2.1 Backend composition

#### Entrypoint và dependency wiring
- File: backend/cmd/main.go
- Chịu trách nhiệm:
  - Parse command-line options.
  - Load config YAML (config.yaml, topics.yaml, sources.yaml).
  - Open DB connection.
  - Wire postgres.Store vào các repository interfaces.
  - Chạy command tương ứng: migrate, schedule, worker, list, show, weak_label, train, select, predict, api.

#### API module
- Files:
  - backend/internal/application/api/server.go
  - backend/internal/application/api/handlers.go
- Routing hiện có:
  - GET /api/topics
  - GET /api/documents?topic=&limit=
  - GET /api/documents/{id}
  - POST /api/schedule
  - GET /api/workflows?limit=
  - GET /api/workflows/{id}/steps
- CORS: allow all origin cho local dev.
- Static serve: route / trỏ tới ../frontend/dist.

#### Schedule -> Learning orchestration qua API
- POST /api/schedule không chỉ discovery.
- Sau khi discovery thành công, backend chạy post-schedule pipeline:
  1. Worker step (timeout 180s).
  2. WeakLabel step.
  3. Train step.
  4. Select step.
  5. Predict step.
- Mục tiêu: 1 nút Run Schedule ở frontend có thể kích hoạt full cycle.

### 2.2 Domain and repositories

- Domain models: document, learning, queue, workflow.
- Repository contracts đặt tại backend/internal/domain/repository/repository.go.
- postgres.Store implement các contracts:
  - BootstrapRepository
  - QueueRepository
  - DocumentRepository
  - CrawlWriteRepository
  - LearningRepository
  - ModelRepository
  - MigrationRepository
  - WorkflowRepository

### 2.3 Discovery design

- Discovery sources gồm RSS và Sitemap.
- URL normalize trước khi enqueue.
- Topic filtering theo source/topic config và heuristic theo từng nhóm topic.
- Output của discovery ghi vào crawl_queue với priority.

### 2.4 Worker design

Pipeline mỗi queue item:
1. Dequeue batch từ DB (lock-safe).
2. Fetch HTML.
3. Extract title/content/metadata.
4. Rule-based classify.
5. Quality gate nội dung.
6. Upsert documents.
7. Mark done/fail và retry theo policy.

### 2.5 Learning/ML design

ML stack (backend/internal/infrastructure/machine_learning):
- TF-IDF vectorizer.
- Logistic Regression.
- Model bundle codec.
- Batch selection cân bằng cho active learning.

Use-cases learning:
- weak_label: sinh nhãn yếu cho tài liệu chưa gán.
- train: train model từ gold + weak labels theo confidence threshold.
- select: chọn mẫu uncertain/diverse cho label_queue.
- predict: ghi ml_topic/ml_confidence/ml_scores trở lại documents.

### 2.6 Workflow persistence

- Workflow executions và step executions được lưu DB.
- API frontend có thể truy vấn list workflows và step logs để theo dõi quá trình chạy.

### 2.7 Frontend design

- Framework: React + TypeScript + Vite.
- Main modules:
  - Documents page: list/filter/detail, trigger schedule.
  - Topics page: taxonomy view.
  - Workflows page: lịch sử execution.
  - Workflow steps page: logs theo workflow.
- API client tách riêng trong frontend/src/api/client.ts.

### 2.8 Security/operations notes (Windows Application Control)

Trong môi trường có AppLocker/WDAC:
- Có thể bị chặn go run ./cmd do Go tạo binary tạm trong AppData.
- Có thể bị chặn native Node addon (rollup.win32-x64-msvc.node).

Khuyến nghị run commands:

```bash
# Backend (policy-friendly)
cd backend
go run ./cmd/main.go api --config ./config/config.yaml --addr :8080

# Hoặc build binary cố định
cd backend
go build -o ./bin/agent-crawl.exe ./cmd
./bin/agent-crawl.exe api --config ./config/config.yaml --addr :8080

# Frontend
cd frontend
npm install
npm run dev
```

## 3. Feature matrix hiện có

| Domain | Feature | Status | Notes |
|---|---|---|---|
| Discovery | RSS discovery | Done | Enqueue URL từ RSS sources |
| Discovery | Sitemap discovery | Done | Hỗ trợ sitemap/urlset, giới hạn theo config |
| Queue | Retry/backoff | Done | attempts + next_run_at + max attempts |
| Processing | HTML fetch + extract | Done | Trích xuất title/content/metadata |
| Processing | Rule-based topic classification | Done | Dựa trên topics/keywords config |
| Persistence | Upsert document + dedupe | Done | Theo canonical URL/content hash |
| Learning | Weak labeling | Done | labels_weak pipeline |
| Learning | Model training | Done | TF-IDF + Logistic Regression |
| Learning | Active learning selection | Done | Chọn batch cho label queue |
| Learning | Prediction write-back | Done | Cập nhật ml_* columns trên documents |
| Workflow | Workflow execution persistence | Done | Lưu workflow/steps vào DB |
| API | Document/topic/workflow endpoints | Done | Expose qua net/http mux |
| API | Trigger full schedule pipeline | Done | POST /api/schedule có post-schedule steps |
| Frontend | Documents dashboard | Done | List/filter/detail + run schedule |
| Frontend | Workflow monitor UI | Done | Workflow list + steps |
| Frontend | Topics UI | Done | Danh sách taxonomy |
| Deployment | Serve frontend dist từ backend | Done | Route / -> ../frontend/dist |

## 4. Roadmap ngắn hạn

- Thêm auth/authz cho API trước khi public deployment.
- Bổ sung metrics và health endpoints (queue lag, success rate, step duration).
- Tối ưu hóa observability: trace theo workflow_id.
- Bổ sung integration tests cho API và orchestrator pipeline.
- Tách scheduler và worker thành process độc lập nếu cần scale lớn.

## 5. Quick start

### 5.1 Backend

```bash
cd backend
go run ./cmd/main.go migrate --config ./config/config.yaml
go run ./cmd/main.go api --config ./config/config.yaml --addr :8080
```

### 5.2 Frontend (dev)

```bash
cd frontend
npm install
npm run dev
```

### 5.3 Frontend production build

```bash
cd frontend
npm run build
```

Sau khi build, backend sẽ phục vụ nội dung từ frontend/dist qua route /.
