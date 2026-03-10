import React, { useRef, useState } from 'react'
import { toast } from 'react-hot-toast'
import HelpTooltip from './HelpTooltip'

interface DockerfileInputProps {
    value: {
        source: 'path' | 'content' | 'upload'
        path?: string
        content?: string
        filename?: string
    }
    onChange: (value: {
        source: 'path' | 'content' | 'upload'
        path?: string
        content?: string
        filename?: string
    }) => void
    error?: string
    disabled?: boolean
}

const DOCKERFILE_TEMPLATES = {
    'Base Linux': `FROM ubuntu:22.04

# Install basic tools
RUN apt-get update && apt-get install -y \\
    curl \\
    wget \\
    git \\
    vim \\
    build-essential \\
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy application files
COPY . .

# Default command
CMD ["bash"]`,

    'Python': `FROM python:3.11-slim

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1 \\
    PYTHONUNBUFFERED=1 \\
    PIP_NO_CACHE_DIR=1 \\
    PIP_DISABLE_PIP_VERSION_CHECK=1

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \\
    build-essential \\
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

# Expose port
EXPOSE 8000

# Run the application
CMD ["python", "app.py"]`,

    'Node.js': `FROM node:18-alpine

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy application code
COPY . .

# Expose port
EXPOSE 3000

# Start the application
CMD ["npm", "start"]`,

    'Java': `FROM openjdk:17-jdk-slim

# Set working directory
WORKDIR /app

# Copy Maven/Gradle files
COPY pom.xml ./
COPY src ./src/

# Build the application
RUN ./mvnw clean package -DskipTests

# Copy built artifact
COPY target/*.jar app.jar

# Expose port
EXPOSE 8080

# Run the application
CMD ["java", "-jar", "app.jar"]`,

    'Go': `FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]`,

    'Rust': `FROM rust:1.70-slim AS builder

# Set working directory
WORKDIR /app

# Copy Cargo files
COPY Cargo.toml Cargo.lock ./

# Create dummy main.rs to cache dependencies
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release
RUN rm -rf src

# Copy source code
COPY src ./src

# Build the application
RUN cargo build --release

# Final stage
FROM debian:bullseye-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \\
    ca-certificates \\
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/target/release/app .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./app"]`
}

export const DockerfileInput: React.FC<DockerfileInputProps> = ({
    value,
    onChange,
    error,
    disabled = false
}) => {
    const [activeTab, setActiveTab] = useState<'path' | 'content' | 'upload'>(value.source || 'path')
    const [selectedTemplate, setSelectedTemplate] = useState<string>('')
    const fileInputRef = useRef<HTMLInputElement>(null)

    const handleTabChange = (tab: 'path' | 'content' | 'upload') => {
        setActiveTab(tab)
        onChange({
            ...value,
            source: tab
        })
    }

    const handlePathChange = (path: string) => {
        onChange({
            source: 'path',
            path
        })
    }

    const handleContentChange = (content: string) => {
        onChange({
            source: 'content',
            content
        })
    }

    const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
        const file = event.target.files?.[0]
        if (!file) return

        // Check file size (1MB limit)
        if (file.size > 1024 * 1024) {
            toast.error('Dockerfile must be less than 1MB')
            return
        }

        // Check file type
        const allowedTypes = ['text/plain', 'application/octet-stream', '']
        if (!allowedTypes.includes(file.type) && !file.name.toLowerCase().includes('dockerfile')) {
            toast.error('Please select a valid Dockerfile')
            return
        }

        const reader = new FileReader()
        reader.onload = (e) => {
            const content = e.target?.result as string
            onChange({
                source: 'upload',
                content,
                filename: file.name
            })
            toast.success(`Uploaded ${file.name}`)
        }
        reader.readAsText(file)
    }

    const handleTemplateSelect = (templateName: string) => {
        const template = DOCKERFILE_TEMPLATES[templateName as keyof typeof DOCKERFILE_TEMPLATES]
        if (template) {
            handleContentChange(template)
            setSelectedTemplate(templateName)
            toast.success(`Applied ${templateName} template`)
        }
    }

    const validateDockerfile = (content: string): string[] => {
        const errors: string[] = []

        if (!content.trim()) {
            errors.push('Dockerfile cannot be empty')
            return errors
        }

        const lines = content.split('\n')
        let hasFrom = false

        for (let i = 0; i < lines.length; i++) {
            const line = lines[i].trim()
            if (!line || line.startsWith('#')) continue

            if (line.toUpperCase().startsWith('FROM ')) {
                hasFrom = true
                break
            }
        }

        if (!hasFrom) {
            errors.push('Dockerfile must start with a FROM instruction')
        }

        return errors
    }

    const getValidationErrors = () => {
        if (activeTab === 'content' && value.content) {
            return validateDockerfile(value.content)
        }
        if (activeTab === 'upload' && value.content) {
            return validateDockerfile(value.content)
        }
        return []
    }

    const validationErrors = getValidationErrors()

    return (
        <div className="space-y-4">
            <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300">
                Dockerfile *
            </label>

            {/* Tab Navigation */}
            <div className="border-b border-gray-200 dark:border-gray-700">
                <nav className="-mb-px flex space-x-4">
                    {[
                        { id: 'path', label: 'File Path', icon: '📁' },
                        { id: 'content', label: 'Paste Content', icon: '📝' },
                        { id: 'upload', label: 'Upload File', icon: '📤' }
                    ].map((tab) => (
                        <button
                            key={tab.id}
                            onClick={() => handleTabChange(tab.id as any)}
                            disabled={disabled}
                            className={`py-2 px-1 border-b-2 font-medium text-sm ${activeTab === tab.id
                                ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                                : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600'
                                } ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
                        >
                            <span className="mr-2">{tab.icon}</span>
                            {tab.label}
                        </button>
                    ))}
                </nav>
            </div>

            {/* Tab Content */}
            <div className={activeTab === 'path' ? '' : 'min-h-[200px]'}>
                {activeTab === 'path' && (
                    <div className="space-y-2">
                        <input
                            type="text"
                            value={value.path || ''}
                            onChange={(e) => handlePathChange(e.target.value)}
                            placeholder="Dockerfile"
                            disabled={disabled}
                            className={`w-full px-3 py-2 border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:border-gray-600 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400 ${error ? 'border-red-300 dark:border-red-600' : 'border-gray-300'
                                } ${disabled ? 'bg-gray-100 dark:bg-gray-800 cursor-not-allowed' : ''}`}
                        />
                        <p className="text-xs text-gray-500 dark:text-gray-400">
                            Path to Dockerfile within build context (e.g., "Dockerfile", "docker/prod.Dockerfile")
                            <HelpTooltip
                                className="ml-2 align-middle"
                                text='Dockerfile and COPY/ADD paths are resolved from the Build Context root. Example: if Dockerfile uses "COPY cmd ./cmd", set Build Context to "examples/image-factory-user-docs" when "cmd/" exists there.'
                            />
                        </p>
                    </div>
                )}

                {activeTab === 'content' && (
                    <div className="space-y-3">
                        {/* Template Selector */}
                        <div className="flex items-center space-x-2">
                            <label className="text-sm text-gray-600 dark:text-gray-400">Template:</label>
                            <select
                                value={selectedTemplate}
                                onChange={(e) => handleTemplateSelect(e.target.value)}
                                disabled={disabled}
                                className="px-2 py-1 border border-gray-300 dark:border-gray-600 rounded text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 dark:bg-gray-800 dark:text-white dark:focus:ring-blue-400"
                            >
                                <option value="">Choose a template...</option>
                                {Object.keys(DOCKERFILE_TEMPLATES).map((template) => (
                                    <option key={template} value={template}>
                                        {template}
                                    </option>
                                ))}
                            </select>
                        </div>

                        {/* Content Textarea */}
                        <textarea
                            value={value.content || ''}
                            onChange={(e) => handleContentChange(e.target.value)}
                            placeholder={`FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \\
    curl \\
    git \\
    && rm -rf /var/lib/apt/lists/*

COPY . /app
WORKDIR /app

CMD ["bash"]`}
                            rows={12}
                            disabled={disabled}
                            className={`w-full px-3 py-2 border rounded-md text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:border-gray-600 dark:text-white dark:placeholder-gray-400 dark:focus:ring-blue-400 ${error || validationErrors.length > 0 ? 'border-red-300 dark:border-red-600' : 'border-gray-300'
                                } ${disabled ? 'bg-gray-100 dark:bg-gray-800 cursor-not-allowed' : ''}`}
                            style={{ fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace' }}
                        />

                        {/* Content Info */}
                        <div className="flex justify-between text-xs text-gray-500 dark:text-gray-400">
                            <span>{(value.content || '').length} characters</span>
                            <span>{(value.content || '').split('\n').length} lines</span>
                        </div>
                    </div>
                )}

                {activeTab === 'upload' && (
                    <div className="space-y-3">
                        <div className="border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-6 text-center">
                            <input
                                ref={fileInputRef}
                                type="file"
                                accept=".dockerfile,text/plain,application/octet-stream"
                                onChange={handleFileUpload}
                                disabled={disabled}
                                className="hidden"
                            />

                            {value.filename ? (
                                <div className="space-y-2">
                                    <div className="text-green-600 dark:text-green-400 text-lg">✓</div>
                                    <p className="text-sm text-gray-700 dark:text-gray-300">{value.filename}</p>
                                    <p className="text-xs text-gray-500 dark:text-gray-400">
                                        {(value.content || '').length} characters
                                    </p>
                                    <button
                                        onClick={() => fileInputRef.current?.click()}
                                        disabled={disabled}
                                        className="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 text-sm underline"
                                    >
                                        Replace file
                                    </button>
                                </div>
                            ) : (
                                <div className="space-y-2">
                                    <div className="text-gray-400 dark:text-gray-500 text-2xl">📄</div>
                                    <p className="text-sm text-gray-600 dark:text-gray-400">
                                        Drag & drop a Dockerfile or click to browse
                                    </p>
                                    <button
                                        onClick={() => fileInputRef.current?.click()}
                                        disabled={disabled}
                                        className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 dark:bg-blue-700 dark:hover:bg-blue-600 text-sm disabled:opacity-50"
                                    >
                                        Choose File
                                    </button>
                                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                                        Max size: 1MB • Supports: Dockerfile, .txt, plain text
                                    </p>
                                </div>
                            )}
                        </div>
                    </div>
                )}
            </div>

            {/* Validation Errors */}
            {
                (error || validationErrors.length > 0) && (
                    <div className="space-y-1">
                        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}
                        {validationErrors.map((err, index) => (
                            <p key={index} className="text-sm text-red-600 dark:text-red-400">• {err}</p>
                        ))}
                    </div>
                )
            }

            {/* Help Text */}
            <p className="text-xs text-gray-500 dark:text-gray-400">
                {activeTab === 'path' && 'Reference an existing Dockerfile in your repository'}
                {activeTab === 'content' && 'Write or paste your Dockerfile content directly'}
                {activeTab === 'upload' && 'Upload a Dockerfile from your local machine'}
            </p>
        </div>
    )
}

export default DockerfileInput
