# 🎬 GoStream

<div align="center">

![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![gRPC](https://img.shields.io/badge/gRPC-244c5a?style=for-the-badge&logo=google&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-316192?style=for-the-badge&logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-DC382D?style=for-the-badge&logo=redis&logoColor=white)
![MinIO](https://img.shields.io/badge/MinIO-C72E49?style=for-the-badge&logo=minio&logoColor=white)
![RabbitMQ](https://img.shields.io/badge/RabbitMQ-FF6600?style=for-the-badge&logo=rabbitmq&logoColor=white)
![FFmpeg](https://img.shields.io/badge/FFmpeg-007808?style=for-the-badge&logo=ffmpeg&logoColor=white)

**A modern, scalable video streaming platform built with Go**

[Features](#-features) • [Architecture](#-architecture) • [Getting Started](#-getting-started) • [API](#-api) • [Roadmap](#-roadmap)

</div>

---

## 🌟 Overview

GoStream is a high-performance video streaming service that handles video upload, transcoding, and adaptive bitrate streaming (HLS). Built with clean architecture principles and designed for scalability.

```
📹 Upload → 🔄 Transcode → 📡 Stream → 🎉 Enjoy!
```

---

## ✨ Features

### 🎥 Core Streaming

- **📤 Video Upload** — Secure presigned URL uploads directly to object storage
- **🔄 Automatic Transcoding** — FFmpeg-powered HLS conversion with multiple quality levels
- **📡 Adaptive Streaming** — HLS protocol for smooth playback across devices
- **🔒 Secure Streaming** — Token-based authentication for video access

### 👤 User Management

- **🔐 JWT Authentication** — Secure access & refresh token system
- **📝 User Registration** — Account creation with validation
- **🔑 Password Management** — Change password & reset via email token
- **👤 Profile Management** — Update user details

### 🏗️ Infrastructure

- **⚡ gRPC + REST** — High-performance gRPC with REST gateway
- **📊 Background Processing** — Async video processing via message queue
- **💾 Object Storage** — S3-compatible storage with MinIO
- **🗄️ Relational Database** — PostgreSQL with GORM ORM

### 🚀 Coming Soon

- **🤖 AI-Powered Recommendations** — Personalized video suggestions based on watch history
- **🔍 RAG Search** — Retrieval-Augmented Generation for intelligent video search
- **📊 Analytics Dashboard** — View counts, watch time, engagement metrics
- **💬 Comments & Reactions** — Social features for video engagement
- **📱 Mobile SDKs** — iOS and Android client libraries

---

## 🏛️ Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Apps                              │
│                    (Web, Mobile, Desktop)                        │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                      API Gateway (:8080)                         │
│                   gRPC-Gateway (REST → gRPC)                     │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                     gRPC Server (:50051)                         │
│              ┌─────────────┬─────────────┐                       │
│              │ AuthService │VideoService │                       │
│              └─────────────┴─────────────┘                       │
└──────────────────────────┬──────────────────────────────────────┘
                           │
          ┌────────────────┼────────────────┐
          ▼                ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  PostgreSQL  │  │    MinIO     │  │   RabbitMQ   │
│   (Users,    │  │   (Videos,   │  │   (Video     │
│   Videos)    │  │    HLS)      │  │  Processing) │
└──────────────┘  └──────────────┘  └──────┬───────┘
                                          │
                                          ▼
                                 ┌──────────────┐
                                 │   Worker     │
                                 │  (FFmpeg     │
                                 │  Transcoder) │
                                 └──────────────┘
```

---

## 🛠️ Tech Stack

| Category        | Technology                                                                                                | Purpose                        |
| --------------- | --------------------------------------------------------------------------------------------------------- | ------------------------------ |
| **Language**    | ![Go](https://img.shields.io/badge/-Go-00ADD8?style=flat&logo=go&logoColor=white)                         | Backend server                 |
| **API**         | ![gRPC](https://img.shields.io/badge/-gRPC-244c5a?style=flat&logo=google&logoColor=white)                 | Service communication          |
| **Gateway**     | gRPC-Gateway                                                                                              | REST API exposure              |
| **Database**    | ![PostgreSQL](https://img.shields.io/badge/-PostgreSQL-316192?style=flat&logo=postgresql&logoColor=white) | Primary data store             |
| **Cache**       | ![Redis](https://img.shields.io/badge/-Redis-DC382D?style=flat&logo=redis&logoColor=white)                | Session & caching              |
| **Storage**     | ![MinIO](https://img.shields.io/badge/-MinIO-C72E49?style=flat&logo=minio&logoColor=white)                | Object storage (S3-compatible) |
| **Queue**       | ![RabbitMQ](https://img.shields.io/badge/-RabbitMQ-FF6600?style=flat&logo=rabbitmq&logoColor=white)       | Message broker                 |
| **Transcoding** | ![FFmpeg](https://img.shields.io/badge/-FFmpeg-007808?style=flat&logo=ffmpeg&logoColor=white)             | Video processing               |
| **ORM**         | GORM                                                                                                      | Database operations            |
| **Validation**  | go-playground/validator                                                                                   | Input validation               |
| **Auth**        | JWT                                                                                                       | Token-based authentication     |

---

## 🚀 Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- MinIO
- RabbitMQ
- FFmpeg
- protoc (Protocol Buffers compiler)

### Installation

1️⃣ **Clone the repository**

```bash
git clone https://github.com/hunderaweke/gostream.git
cd gostream
```

2️⃣ **Set up environment variables**

```bash
cp .env.sample .env
# Edit .env with your configuration
```

3️⃣ **Install dependencies**

```bash
go mod download
```

4️⃣ **Start infrastructure services**

```bash
# PostgreSQL
docker run -d --name postgres -p 5432:5432 \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=gostream \
  postgres:18

# MinIO
docker run -d --name minio -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"

# RabbitMQ
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 \
  -e RABBITMQ_DEFAULT_USER=guest \
  -e RABBITMQ_DEFAULT_PASS=guest \
  rabbitmq:3-management
```

5️⃣ **Generate protobuf code**

```bash
protoc -Iinternal/proto -Ithird_party \
  --go_out=gen/go --go_opt=module=github.com/hunderaweke/gostream/gen/go \
  --go-grpc_out=gen/go --go-grpc_opt=module=github.com/hunderaweke/gostream/gen/go \
  --grpc-gateway_out=gen/go --grpc-gateway_opt=module=github.com/hunderaweke/gostream/gen/go \
  internal/proto/*.proto
```

6️⃣ **Run the server**

```bash
go run cmd/api/main.go
```

---

## 📡 API

### 🔐 Authentication

| Method | Endpoint                   | Description          |
| ------ | -------------------------- | -------------------- |
| `POST` | `/v1/auth/register`        | Register new user    |
| `POST` | `/v1/auth/login`           | Login & get tokens   |
| `POST` | `/v1/auth/refresh`         | Refresh access token |
| `POST` | `/v1/auth/change-password` | Change password      |
| `POST` | `/v1/auth/reset-password`  | Reset password       |

### 🎥 Videos

| Method | Endpoint                   | Description                   |
| ------ | -------------------------- | ----------------------------- |
| `POST` | `/v1/videos`               | Create video & get upload URL |
| `POST` | `/v1/videos/{id}/complete` | Mark upload complete          |
| `GET`  | `/v1/videos`               | List videos (paginated)       |
| `GET`  | `/v1/videos/{id}`          | Get video details             |
| `GET`  | `/v1/stream/{id}`          | Stream video (HLS)            |

### Example: Upload a Video

```bash
# 1. Create video record & get presigned upload URL
curl -X POST http://localhost:8080/v1/videos \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"title": "My Video", "description": "A cool video"}'

# Response: { "video_id": "abc-123", "upload_url": "http://..." }

# 2. Upload file to presigned URL
curl -X PUT "<upload_url>" \
  -H "Content-Type: video/mp4" \
  --data-binary @video.mp4

# 3. Mark upload complete (triggers transcoding)
curl -X POST http://localhost:8080/v1/videos/abc-123/complete \
  -H "Authorization: Bearer <token>"

# 4. Stream the video (after transcoding)
curl http://localhost:8080/v1/stream/abc-123
```

---

## 📁 Project Structure

```
gostream/
├── 📂 cmd/
│   └── 📂 api/
│       └── 📄 main.go              # Application entry point
├── 📂 gen/
│   └── 📂 go/                      # Generated protobuf code
├── 📂 internal/
│   ├── 📂 database/                # Database connections
│   │   ├── 📄 postgres.go
│   │   ├── 📄 redis.go
│   │   └── 📄 minio.go
│   ├── 📂 domain/                  # Business entities & interfaces
│   │   ├── 📄 user.go
│   │   ├── 📄 video.go
│   │   └── 📄 model.go
│   ├── 📂 grpc_server/             # gRPC service implementations
│   │   ├── 📄 auth.go
│   │   └── 📄 video.go
│   ├── 📂 proto/                   # Protocol buffer definitions
│   │   ├── 📄 auth.proto
│   │   └── 📄 video.proto
│   ├── 📂 queue/                   # Message queue handlers
│   ├── 📂 repository/              # Data access layer
│   ├── 📂 server/handlers/         # HTTP handlers
│   └── 📂 usecase/                 # Business logic
├── 📂 pkg/
│   ├── 📂 interceptors/            # gRPC interceptors
│   └── 📂 utils/                   # Utilities (JWT, etc.)
├── 📂 third_party/                 # External proto files
├── 📄 .env.sample
├── 📄 go.mod
└── 📄 README.md
```

---

## 🗺️ Roadmap

### Phase 1: Core Platform ✅

- [x] User authentication (JWT)
- [x] Video upload with presigned URLs
- [x] HLS transcoding pipeline
- [x] Secure video streaming
- [x] RESTful API via gRPC-Gateway

### Phase 2: Enhanced Features 🚧

- [ ] 📊 View count & analytics
- [ ] 💬 Comments & reactions
- [ ] 🏷️ Video tags & categories
- [ ] 🔍 Full-text search
- [ ] 📱 Mobile-friendly API

### Phase 3: AI & Personalization 🔮

- [ ] 🤖 **RAG-powered Search** — Semantic video search using embeddings
- [ ] 🎯 **Smart Recommendations** — ML-based suggestions from watch history

### Phase 4: Scale & Enterprise 🚀

- [ ] 🌍 CDN integration
- [ ] 📈 Horizontal scaling
- [ ] 🔐 Enterprise SSO
- [ ] 📊 Admin dashboard
- [ ] 💰 Monetization features

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 👨‍💻 Author

**Hundera Awoke**

- GitHub: [@hunderaweke](https://github.com/hunderaweke)

---

<div align="center">

**⭐ Star this repo if you find it useful! ⭐**

Made with ❤️ and Go

</div>
