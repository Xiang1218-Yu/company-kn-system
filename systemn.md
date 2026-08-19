# 需求文档一：企业级知识库问答系统

## 1. 项目概述

### 1.1 项目名称
企业级知识库问答系统（Enterprise Knowledge Q&A System）

### 1.2 项目目标
构建一个企业内部知识管理与智能问答平台，帮助团队沉淀、检索和复用知识资产，减少重复性咨询，提升信息获取效率。

### 1.3 目标用户
- 企业员工（普通用户）：查找公司制度、技术文档、项目经验
- 知识管理员：上传、分类、审核知识文档
- 系统管理员：用户管理、权限配置、系统监控

### 1.4 核心业务流程
文档上传 → 解析与切片 → 向量化存储 → 用户提问 → 语义检索 → LLM生成答案 → 用户反馈
---

## 2. 功能模块

### 2.1 用户与权限模块
- 用户注册/登录（邮箱+密码，JWT认证）
- 用户角色：普通成员、知识管理员、系统管理员
- 知识库权限隔离：按团队/部门划分独立知识空间
- 用户加入知识库需邀请或申请审批

### 2.2 文档管理模块
- 支持上传格式：PDF、Word（.docx）、Markdown（.md）、纯文本（.txt）
- 支持批量上传（拖拽或选择文件夹）
- 支持文档分类（按部门/项目/文档类型）
- 支持文档标签（手动添加或自动提取）
- 文档版本管理：覆盖上传时保留历史版本
- 文档状态：待处理/已索引/索引失败

### 2.3 文档解析与索引模块
- 后端自动解析文档内容（提取纯文本、表格、标题层级）
- 按语义段落或固定长度切分文档
- 调用向量化接口生成文本向量并存储
- 索引进度实时反馈给前端
- 索引失败时记录错误日志并支持重试

### 2.4 智能问答模块
- 用户输入自然语言问题
- 系统进行语义检索，召回相关文档片段（Top 5-10）
- 调用大模型API（OpenAI或国产LLM）生成答案
- 答案附带来源引用（文档标题+原文片段高亮）
- 支持追问（上下文连续对话）
- 支持答案复制、点赞/点踩反馈
- 问题无答案时提示“知识库中暂无相关信息”

### 2.5 检索与浏览模块
- 关键词全文搜索（标题+正文）
- 按分类/标签/时间筛选文档列表
- 文档详情页查看原文内容和元数据
- 文档下载权限控制

### 2.6 反馈与运营模块
- 用户对答案的点赞/点踩记录
- 常见问题（FAQ）自动归纳（高频问题+最佳答案）
- 文档更新通知（订阅者接收变更提醒）
- 运营看板：文档数、问答量、满意率

---

## 3. 非功能性需求

### 3.1 性能要求
- 文档上传响应时间 < 3秒
- 索引处理支持异步队列，不阻塞用户操作
- 问答响应时间 < 5秒（含LLM调用）
- 全文检索响应时间 < 1秒
- 支持并发用户数：初期100，可水平扩展

### 3.2 安全要求
- JWT Token 过期时间可配置（默认24小时）
- 文档存储加密（传输加密 TLS 1.2+）
- 操作日志记录（上传/删除/查询等关键操作）
- 用户密码加密存储（bcrypt）
- 防SQL注入、XSS攻击

### 3.3 可用性要求
- 系统可用率 ≥ 99.5%
- 支持优雅停机（正在处理的请求不中断）
- LLM调用失败时降级返回“服务繁忙，请稍后重试”
- 索引失败自动重试3次

### 3.4 扩展性要求
- 支持对接多种LLM（OpenAI/Claude/国产大模型）
- 向量存储支持切换（pgvector/Milvus/Elasticsearch）
- 文件存储支持切换（本地/MinIO/S3/OSS）

---

## 4. 技术栈

| 层级 | 技术选型 |
|------|----------|
| 后端框架 | Gin |
| 数据库 | PostgreSQL + pgvector扩展 |
| 缓存 | Redis |
| 文件存储 | MinIO / 阿里云OSS |
| 向量模型 | text-embedding-ada-002 / BGE |
| LLM | OpenAI API / 国产大模型 |
| 任务队列 | RabbitMQ / Redis Stream |
| 前端框架 | React + TypeScript + Ant Design |
| 认证授权 | JWT |
| 日志 | Zap |
| 配置管理 | Viper |

---

## 5. 数据库设计（核心表）

### 5.1 users（用户表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| email | VARCHAR(255) | 邮箱（唯一） |
| password_hash | VARCHAR(255) | 密码哈希 |
| name | VARCHAR(100) | 用户姓名 |
| role | VARCHAR(50) | admin/manager/member |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### 5.2 knowledge_bases（知识库表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| name | VARCHAR(100) | 知识库名称 |
| description | TEXT | 描述 |
| team_id | UUID | 所属团队ID |
| created_by | UUID | 创建人ID |
| created_at | TIMESTAMP | 创建时间 |

### 5.3 documents（文档表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| kb_id | UUID | 所属知识库ID |
| name | VARCHAR(255) | 文档名称 |
| file_path | VARCHAR(500) | 文件存储路径 |
| file_size | BIGINT | 文件大小（字节） |
| file_type | VARCHAR(20) | pdf/docx/md/txt |
| status | VARCHAR(20) | pending/indexing/indexed/failed |
| chunk_count | INT | 切片数量 |
| uploaded_by | UUID | 上传人ID |
| created_at | TIMESTAMP | 上传时间 |
| updated_at | TIMESTAMP | 更新时间 |

### 5.4 chunks（切片表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| doc_id | UUID | 所属文档ID |
| content | TEXT | 切片文本内容 |
| vector | vector(1536) | 文本向量 |
| chunk_index | INT | 切片序号 |
| metadata | JSONB | 元数据（页码/标题等） |

### 5.5 qa_logs（问答日志表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 用户ID |
| kb_id | UUID | 知识库ID |
| question | TEXT | 用户问题 |
| answer | TEXT | 系统回答 |
| sources | JSONB | 引用来源列表 |
| feedback | VARCHAR(20) | 点赞/点踩/无 |
| response_time | INT | 响应耗时（毫秒） |
| created_at | TIMESTAMP | 提问时间 |

---

## 6. API接口设计（核心）

### 6.1 文档管理
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/kb/:kbId/documents | 上传文档 |
| GET | /api/v1/kb/:kbId/documents | 获取文档列表 |
| GET | /api/v1/documents/:id | 获取文档详情 |
| DELETE | /api/v1/documents/:id | 删除文档 |
| GET | /api/v1/documents/:id/status | 获取索引状态 |

### 6.2 智能问答
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/qa/ask | 提问（SSE流式响应） |
| GET | /api/v1/qa/history | 获取问答历史 |
| POST | /api/v1/qa/feedback | 提交反馈 |

### 6.3 检索浏览
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/search?q=&kbId= | 全文/语义搜索 |
| GET | /api/v1/documents/:id/download | 下载文档 |

### 6.4 知识库管理
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/kb | 创建知识库 |
| GET | /api/v1/kb | 获取知识库列表 |
| PUT | /api/v1/kb/:id | 更新知识库 |
| DELETE | /api/v1/kb/:id | 删除知识库 |
| POST | /api/v1/kb/:id/invite | 邀请成员 |

---


