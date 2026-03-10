# Build Pipeline Journey

This document walks through the end-to-end build pipeline journey in Image Factory, from build submission through execution and artifact delivery.

## Overview

The build pipeline in Image Factory transforms your code into deployable artifacts through a comprehensive, automated process. This journey covers everything from initial build submission to final artifact delivery, including monitoring, error handling, and optimization.

### Key Concepts

- **Build Methods**: Docker, Buildx, Kaniko, Packer, Nix, Paketo
- **Pipeline Stages**: Submission → Validation → Execution → Completion → Delivery
- **Artifacts**: Container images, machine images, packages
- **Monitoring**: Real-time status, logs, metrics, notifications

---

## 📁 Phase 0: Project Setup

### 0.1 Choose Your Project Creation Method

#### **Quick Start (For Later Repository Setup)**
```
🎯 When to use: You want to create the project now, configure repository later
📍 How to access: Projects page → Click "Create Project" button
📝 What you'll enter: Project name, description, visibility only
⚠️  Important: You'll need to add repository info before creating builds
```

#### **Complete Setup (Ready for Builds Immediately)**
```
🎯 When to use: You have repository details ready and want to build right away
📍 How to access: Projects page → Click "New Project" link (not button)
📝 What you'll enter: Project name, description, repository URL, default branch
✅ Ready: Can create builds immediately after creation
```

### 0.2 How to Tell Which Method You're Using

#### **You're Using the Modal (Quick Method):**
- Small popup window appears
- Only 3 fields: Name, Description, Visibility
- No repository URL field visible

#### **You're Using the Full Page (Complete Method):**
- Navigates to dedicated project creation page
- 4+ fields including Repository URL and Branch
- Full form layout with additional options

### 0.3 Check If Project Has Repository Configured

#### **Visual Status Indicators:**
- **Header Badge**: Green "Repository Connected" or Yellow "Repository Not Set" badge next to project name
- **Status Card**: Large colored card below project header showing repository status
- **Branch Display**: Current default branch shown prominently when configured

#### **Method 1: Project Dashboard (Most Obvious)**
```
📍 Look at: Top of project page, right next to project name
👀 Status Badge: "Repository Connected" (green) or "Repository Not Set" (yellow)
✅ Configured: Green badge + detailed repository card below
❌ Not Configured: Yellow badge + warning card with "Configure Repository" button
```

#### **Method 2: Repository Details Card**
```
📍 Look at: Large colored card below project header
👀 What's shown: Repository URL, branch, connection status
✅ Configured: Green card with full repository details
❌ Not Configured: Yellow warning card with setup instructions
```

### 0.4 Add Repository to Existing Project

**If you used quick creation and need to add repository:**

```
1️⃣ Look for: Yellow "Repository Not Set" badge next to project name
2️⃣ Click: "Configure Repository" button in the warning card below header
3️⃣ Or click: "Edit Project" button to access full settings
4️⃣ Enter: Repository URL (HTTPS/SSH format)
5️⃣ Enter: Default branch (usually "main" or "master")
6️⃣ Click: "Save" to update project
✅ Result: Badge turns green, repository card shows connection details
```

**Alternative access:**
- Click "Edit Project" button in the top-right
- Navigate to project settings manually

### 0.5 Quick Decision Guide

| Situation | Recommended Method | Next Steps |
|-----------|-------------------|------------|
| **New to Image Factory** | Quick creation | Add repository later via settings |
| **Have repo details ready** | Complete setup | Start building immediately |
| **Team collaboration** | Complete setup | Everyone can build right away |
| **Just exploring** | Quick creation | Configure when ready to build |
| **Migrating existing project** | Complete setup | Full configuration from start |

### 0.6 Verify Project Readiness

**Before creating your first build, confirm:**
```
✅ Project exists in your tenant
✅ Repository URL is configured (not empty)
✅ Repository is accessible with your credentials
✅ Default branch exists (usually "main" or "master")
```

---

## 🏁 Phase 1: Build Submission

### 1.1 Access Build Interface
**Location:** Projects → [Project Name] → Builds tab or Create Build page

**What happens:**
- Navigate to your project dashboard
- Click "New Build" or "Create Build" button
- Select build method (Docker/Buildx/Kaniko/Packer/Nix/Paketo)

### 1.2 Configure Build Parameters

#### **Basic Configuration**
```
📝 Build Name: Give your build a descriptive name
🏷️  Tags: Add version tags, environment labels
📦 Build Method: Choose appropriate build technology
```

**Note:** Repository source is configured at the project level. When you select a project, the build will automatically use that project's configured Git repository and default branch.

### 1.3 Method-Specific Configuration

#### **🐳 Docker/Buildx/Kaniko Builds**
```
📄 Dockerfile: Upload file, paste content, or use template
🏗️  Build Context: Repository root or subdirectory
🏷️  Target Stage: Multi-stage build target
🔧 Build Args: Environment variables for build
📋 Secrets: Sensitive values for build process
```

#### **📦 Packer Builds**
```
📋 Template: JSON configuration for machine images
🔧 Variables: Dynamic values for template
🎯 Builders: AWS AMI, GCP Images, Azure VMs, etc.
📤 Post-Processors: Artifact handling and distribution
```

#### **❄️ Nix Builds**
```
📄 Expression: Nix language build definition
🎯 Flake URI: Modern Nix flake reference
🏗️  Attributes: Build targets within flake
💾 Cache: Nix store optimization
```

#### **🛠️ Paketo Builds**
```
🏗️  Builder: Paketo builder image selection
📦 Buildpacks: Language/framework buildpacks
🔧 Environment: Build-time configuration
📋 Bindings: External service connections
```

### 1.4 Infrastructure Selection (NEW - Phase 3)

#### **AI-Powered Recommendations**
```
🧠 Smart Selection: AI analyzes your build requirements
⚡ Auto Recommendation: Suggests optimal infrastructure automatically
🎯 Confidence Score: Shows recommendation certainty (80-95%)
🔄 Alternative Options: Provides fallback choices with reasoning
```

#### **Infrastructure Types Available**
```
☸️  Kubernetes Cluster: Scalable container orchestration (Primary)
🖥️  Build Nodes: Dedicated build servers (Fallback)
🎛️  Auto Selection: Let system choose based on requirements (Recommended)
```

#### **How Infrastructure Selection Works**
```
1️⃣ Analysis: System analyzes build method, size, and requirements
2️⃣ Recommendation: AI suggests best infrastructure with confidence score
3️⃣ User Choice: Accept recommendation or manually select alternative
4️⃣ Execution: Build runs on selected infrastructure automatically
```

#### **Infrastructure Selection UI**
```
📍 Location: "Infrastructure" tab in build creation wizard
⏱️  Loading: Recommendations appear within 5 seconds
👀 Display: Recommended infrastructure with reasoning
🎛️  Controls: Radio buttons for manual selection
💡 Help: Explanations for each infrastructure type
```

#### **Important Security Note**
```
🔒 Admin Configuration: Infrastructure providers are configured by system administrators only
👤 User Selection: Users can only select from pre-configured, available infrastructure
🛡️  Permission Separation: No user access to provider credentials or configuration
```

#### **Specialized Routing Examples**
```
🎯 GPU Workloads: Automatically routed to GPU-enabled clusters
📦 Large Builds: Directed to high-memory build nodes
🚀 Standard Builds: Run on cost-effective Kubernetes pods
🔒 Secure Builds: Use compliance-certified infrastructure
```

### 1.5 Advanced Options

#### **Build Optimization**
```
🚀 Parallel Builds: Multi-platform compilation
💾 Caching: Layer caching for faster builds
🔄 Retry Logic: Automatic failure recovery
⏱️  Timeout: Maximum build duration
```

#### **Security & Compliance**
```
🛡️  SBOM Generation: Software bill of materials
🔍 Vulnerability Scanning: Security analysis
📋 Compliance Checks: Policy enforcement
```

### 1.6 Submit Build
**Action:** Click "Submit Build" or "Start Build"

**Immediate Feedback:**
- Build ID generation (e.g., `build-12345`)
- Initial status: "Queued" or "Pending"
- Estimated queue time display
- Infrastructure assignment confirmation

---

## 🔍 Phase 2: Build Validation & Preparation

### 2.1 Pre-Flight Checks
**Duration:** 10-30 seconds

**Validation Steps:**
```
✅ Repository Access: Verify Git permissions for project's configured repository (fails if no repository configured)
✅ Dockerfile/Packer Template: Syntax validation
✅ Build Context: Path existence check within repository
✅ Required Secrets: Availability verification
✅ Resource Limits: Quota compliance check
```

**Possible Outcomes:**
- ✅ **Validation Passed**: Build moves to queue
- ⚠️ **Validation Failed**: Error message with fix guidance
- 🔄 **Retry Available**: Automatic retry for transient issues
- 🚨 **Repository Not Configured**: Redirect to project settings to add repository information

### 2.2 Resource Allocation
**Duration:** 5-60 seconds (depending on queue)

**Infrastructure Assignment Process:**
```
🎯 Smart Routing: AI-powered infrastructure selection based on build requirements
☸️  Kubernetes Priority: Container builds run on K8s pods by default
🖥️  Build Nodes: Specialized workloads use dedicated build servers
⚖️  Load Balancing: Distribute builds across available infrastructure
🔄 Auto-Scaling: Scale infrastructure capacity based on demand
```

**Resource Allocation Steps:**
```
🏗️  Infrastructure Selection: Choose between Kubernetes or build nodes
💾 Volume Mounting: Attach persistent storage if needed
🌐 Network Setup: Configure VPC, security groups
🔑 Secret Injection: Mount secrets securely
📊 Resource Limits: Apply CPU/memory constraints based on infrastructure
```

### 2.3 Build Environment Setup
**Duration:** 10-45 seconds

**Environment Preparation:**
```
🐳 Base Image Pull: Download specified base images
📦 Dependency Cache: Restore cached dependencies
🔧 Tool Installation: Set up build tools and runtimes
📋 Configuration: Apply build-specific settings
```

---

## ⚙️ Phase 3: Build Execution

### 3.0 Infrastructure Assignment & Routing (NEW - Phase 3)

#### **Smart Infrastructure Routing**
```
🎯 Assignment Logic: Based on build requirements and user selection
☸️  Kubernetes Priority: Primary infrastructure for containerized builds
🖥️  Build Nodes Fallback: Dedicated servers for specialized workloads
⚖️  Load Balancing: Distribute across available infrastructure
🔄 Auto-Scaling: Scale infrastructure based on demand
```

#### **Infrastructure Selection Process**
```
1️⃣ Build Analysis: Evaluate build method, size, and requirements
2️⃣ Infrastructure Matching: Find optimal worker based on capabilities
3️⃣ Resource Reservation: Allocate compute resources
4️⃣ Environment Preparation: Set up build environment on assigned infrastructure
```

#### **Infrastructure Types & Capabilities**
```
☸️  Kubernetes Pods:
   • Container-native execution
   • Auto-scaling and self-healing
   • GPU support for ML workloads
   • Cost-effective for standard builds

🖥️  Build Nodes:
   • Dedicated VM instances
   • High-memory configurations
   • Specialized hardware (GPU, ARM)
   • Compliance-certified environments
```

#### **Routing Decision Factors**
```
📏 Build Size: Large builds → High-memory infrastructure
🎯 Build Type: GPU workloads → GPU-enabled clusters
🔒 Security Level: Compliance builds → Certified infrastructure
⚡ Performance: Time-sensitive → Optimized workers
💰 Cost Optimization: Standard builds → Cost-effective pods
```

### 3.1 Build Phases (Method-Specific)

#### **🐳 Docker Build Execution**
```
📦 Phase 1: Base image download (30-120s)
🏗️  Phase 2: Instruction execution (60-900s)
📋 Phase 3: Metadata generation (10-30s)
🏷️  Phase 4: Image tagging (5-15s)
📤 Phase 5: Registry push (20-180s)
```

#### **📦 Packer Build Execution**
```
🔧 Phase 1: Template validation (5-15s)
🏗️  Phase 2: Builder execution (300-3600s)
📤 Phase 3: Artifact upload (60-600s)
🧹 Phase 4: Cleanup (10-30s)
```

#### **❄️ Nix Build Execution**
```
📄 Phase 1: Expression evaluation (10-60s)
📦 Phase 2: Dependency resolution (30-300s)
🏗️  Phase 3: Package compilation (60-1800s)
💾 Phase 4: Store optimization (15-60s)
```

### 3.2 Real-Time Monitoring

#### **Build Status Indicators**
```
🔄 Queued: Waiting for worker availability
⚙️  Preparing: Environment setup in progress
🏃 Running: Active build execution
📊 Processing: Post-build operations
✅ Completed: Successful build finish
❌ Failed: Build error occurred
⏸️  Paused: Manual intervention required
```

#### **Progress Tracking**
```
📈 Progress Bar: Visual completion percentage
⏱️  Elapsed Time: Current build duration
🎯 Current Step: Active build instruction
📊 Metrics: CPU, memory, disk usage
```

### 3.3 Log Streaming

#### **Log Types Available**
```
📋 Build Logs: Command output and messages
🐛 Debug Logs: Detailed execution tracing
📊 System Logs: Infrastructure events
🚨 Error Logs: Failure diagnostics
```

#### **Log Features**
```
🔍 Real-time streaming
📄 Full log download
🔎 Search and filtering
📊 Log analytics
🔔 Log-based alerts
```

### 3.4 Build Artifacts

#### **Artifact Generation**
```
🏷️  Image Tags: Registry references
📦 Package Files: Downloadable assets
📋 SBOM Documents: Security manifests
📊 Build Reports: Execution summaries
🔐 Signatures: Cryptographic verification
```

#### **Artifact Storage**
```
☁️  Cloud Storage: S3, GCS, Azure Blob
🏢 Registry: Docker Hub, ECR, GCR, Harbor
📦 Local Cache: Worker node storage
🔄 CDN: Global distribution
```

---

## 📊 Phase 4: Build Completion & Analysis

### 4.1 Completion Processing
**Duration:** 10-60 seconds

**Final Steps:**
```
✅ Status Update: Mark build as completed
📊 Metrics Collection: Gather performance data
🧹 Resource Cleanup: Free allocated resources
📋 Report Generation: Create build summary
🔔 Notification Dispatch: Alert stakeholders
```

### 4.2 Build Results Dashboard

#### **Success Metrics**
```
⏱️  Total Duration: End-to-end build time
💾 Artifact Size: Generated package size
🏷️  Tags Applied: Version and metadata tags
📊 Test Results: If applicable
🛡️  Security Score: Vulnerability assessment
```

#### **Quality Metrics**
```
📈 Code Coverage: Test coverage percentage
🐛 Issues Found: Static analysis results
📋 Compliance: Policy check results
🔒 Security: Vulnerability scan results
```

### 4.3 Artifact Distribution

#### **Automatic Deployment**
```
🚀 CD Integration: Trigger downstream deployments
📦 Package Registries: Push to artifact repositories
☁️  Cloud Storage: Upload to object storage
🌐 CDN Updates: Refresh content delivery
```

#### **Manual Distribution**
```
📥 Download Links: Direct artifact access
🔗 Registry URLs: Pull instructions
📋 API Endpoints: Programmatic access
🔄 Webhooks: External system integration
```

---

## 🚨 Phase 5: Error Handling & Recovery

### 5.1 Build Failure Scenarios

#### **Common Failure Types**
```
❌ Build Errors: Code compilation failures
🔍 Test Failures: Unit/integration test errors
🛡️  Security Blocks: Policy violation rejections
⏱️  Timeout: Build duration exceeded limits
💾 Resource Exhaustion: Memory/CPU/disk limits
🌐 Network Issues: Connectivity problems
```

#### **Error Classification**
```
🔴 Critical: Build cannot proceed
🟡 Warning: Non-blocking issues
🟢 Info: Informational messages
🔵 Debug: Detailed troubleshooting data
```

### 5.2 Automatic Recovery

#### **Retry Logic**
```
🔄 Automatic Retries: Transient failure recovery
⏳ Exponential Backoff: Progressive delay increases
🎯 Smart Retry: Context-aware retry decisions
📊 Failure Analytics: Pattern recognition
```

#### **Fallback Strategies**
```
🔀 Alternative Workers: Different machine types
📦 Cached Dependencies: Skip redundant downloads
🚀 Parallel Execution: Distribute across workers
🛠️  Tool Updates: Refresh build environment
```

### 5.3 Manual Intervention

#### **Debug Access**
```
🐛 Debug Session: Interactive troubleshooting
📋 Log Analysis: Detailed error examination
🔧 Environment Access: Direct worker connection
📊 Performance Metrics: Resource usage analysis
```

#### **Resolution Options**
```
🔄 Rebuild: Restart with same configuration
⚙️  Reconfigure: Modify build parameters
🛠️  Environment Fix: Update build environment
📞 Support Request: Escalate to engineering team
```

---

## 📈 Phase 6: Build Analytics & Optimization

### 6.1 Performance Monitoring

#### **Build Metrics**
```
⏱️  Build Duration: Average, median, percentiles
📊 Success Rate: Pass/fail percentages
💾 Resource Usage: CPU, memory, disk consumption
🌐 Network Traffic: Data transfer volumes
```

#### **Trend Analysis**
```
📈 Performance Trends: Duration changes over time
🎯 Success Trends: Reliability improvements
💰 Cost Analysis: Resource usage costs
🔍 Bottleneck Identification: Slowest build steps
```

### 6.2 Build History

#### **Historical Data**
```
📚 Build Timeline: Chronological build list
🔍 Search & Filter: Find specific builds
📊 Comparison: Side-by-side build analysis
📋 Change Tracking: Configuration modifications
```

#### **Build Insights**
```
🎯 Most Active: Frequently built projects
⏱️  Slowest Builds: Performance optimization targets
❌ Failure Patterns: Common error identification
✅ Success Patterns: Best practice identification
```

### 6.3 Optimization Recommendations

#### **Automated Suggestions**
```
💾 Cache Optimization: Improve layer caching
🏗️  Parallel Builds: Multi-platform optimization
📦 Dependency Management: Update strategies
🛠️  Tool Updates: Latest version recommendations
```

#### **Manual Optimizations**
```
🔧 Build Script Tuning: Performance improvements
📋 Configuration Refinement: Parameter optimization
🌐 Network Optimization: Registry mirror usage
💽 Storage Optimization: Artifact size reduction
```

---

## 🔄 Phase 7: Integration & Automation

### 7.1 CI/CD Integration

#### **Webhook Triggers**
```
🔗 Git Push: Automatic build on code changes
🏷️  Tag Creation: Release build triggers
📋 PR Events: Pull request validation
📊 Schedule: Time-based build execution
```

#### **Pipeline Orchestration**
```
🔄 Jenkins Integration: Classic CI/CD
🐙 GitHub Actions: Cloud-native workflows
🏗️  GitLab CI: Integrated DevOps
☁️  Cloud Build: Managed build services
```

### 7.2 API Integration

#### **Programmatic Access**
```
🔧 REST API: Full build lifecycle control
📊 Webhooks: Real-time event notifications
🔌 SDKs: Language-specific client libraries
🤖 ChatOps: Slack/Discord integration
```

#### **External Systems**
```
📋 Jira: Issue tracking integration
📊 Datadog: Monitoring and alerting
☁️  CloudWatch: AWS service integration
📈 Grafana: Dashboard visualization
```

---

## 🎯 Success Criteria

### For Build Submission
- ✅ Build accepted without validation errors
- ✅ Clear progress indicators throughout process
- ✅ Real-time status updates and notifications

### For Build Execution
- ✅ Successful artifact generation
- ✅ Complete log availability and searchability
- ✅ Performance metrics collection and analysis

### For Build Completion
- ✅ Artifact delivery to specified destinations
- ✅ Automated deployment triggers (if configured)
- ✅ Comprehensive build reports and analytics

### For Error Scenarios
- ✅ Clear error messages with actionable guidance
- ✅ Automatic retry mechanisms for transient failures
- ✅ Debug access and troubleshooting tools

---

## 🚀 Quick Actions Reference

### Emergency Actions
```
🛑 Stop Build: Immediate build cancellation
🔄 Restart Build: Re-run with same configuration
🐛 Debug Mode: Enable detailed logging
📞 Support: Contact engineering team
```

### Common Tasks
```
📊 View Logs: Access real-time build output
📦 Download Artifacts: Retrieve build outputs
🔍 Search Builds: Find historical builds
📋 View Reports: Access build analytics
```

### Optimization Tasks
```
💾 Enable Caching: Improve build performance
🏷️  Tag Management: Organize build versions
📊 Monitor Trends: Track build performance
🔧 Update Config: Modify build parameters
```

---

## 📞 Support & Troubleshooting

### Self-Service Resources
```
📚 Documentation: Comprehensive build guides
🎥 Video Tutorials: Visual walkthroughs
🤖 Chat Support: AI-powered assistance
📋 FAQ: Common issue resolution
```

### Escalation Paths
```
1️⃣ Team Lead: Project-specific issues
2️⃣ Engineering: Technical build problems
3️⃣ Infrastructure: Platform and scaling issues
4️⃣ Security: Compliance and vulnerability concerns
```

---

**🎉 Congratulations!** You've completed the comprehensive Build Pipeline Journey. Your builds now follow a robust, monitored, and optimized process from submission to shipment.
