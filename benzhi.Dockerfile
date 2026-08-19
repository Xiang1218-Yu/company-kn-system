# 官方 Go 镜像，自带完整工具链
FROM golang:1.26

# 以 Go 镜像为基础，再装 Node.js，两套工具链都保留
RUN apt-get update && apt-get install -y curl \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# 先复制依赖文件并下载依赖，保证容器内离线可用
COPY go.mod go.sum ./
RUN go mod download

# 前端依赖（frontend/ 目录）
COPY frontend/package*.json ./frontend/
RUN cd frontend && npm install

# 复制所有项目文件
COPY . .

# 预编译，确认基础代码健康
RUN go build ./... && cd frontend && npm run build

# 容器启动后进入 shell，方便模型操作
CMD ["bash"]
