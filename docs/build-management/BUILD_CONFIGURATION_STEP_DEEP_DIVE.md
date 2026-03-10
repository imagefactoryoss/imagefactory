# Build Configuration Step Deep Dive

## Overview

The **Configuration Step** is the 3rd step in the Build Creation Wizard. It's where users configure the specific settings for their chosen build method. This step uses a **tab-based interface** with 4 tabs that organize different configuration aspects.

### Key Principle
**The Build Method tab is NOT a validation tab** - it's the final tab that holds method-specific required fields. Each build method has unique requirements:
- **Kaniko**: Requires full Dockerfile content + build context
- **Buildx**: Requires base image + instructions (plus advanced options)
- **Container**: Requires base image + instructions
- **Paketo**: Requires builder selection
- **Packer**: Requires Packer template JSON

---

## Wizard Flow Recap

```
Step 1: Build Method Selection
    ↓ (Choose Kaniko, Buildx, Container, Paketo, or Packer)
    ↓
Step 2: Tool Selection
    ↓ (Select SBOM, scanning tools, registry, secrets manager)
    ↓
Step 3: Configuration ← YOU ARE HERE
    ├─ Tab 1: Basic (Base Image, Instructions, Tags)
    ├─ Tab 2: Environment (Environment variables)
    ├─ Tab 3: Advanced (Method-specific advanced options)
    └─ Tab 4: Build Method (Method-specific required config)
    ↓
Step 4: Validation & Submit
```

---

## The 4 Configuration Tabs Explained

### Tab 1: Basic

**Purpose**: Collect common build configuration needed across most build methods

**Fields**:
1. **Base Image** (for non-Kaniko builds)
   - Type: Text input
   - Example: `ubuntu:20.04`, `node:18-alpine`, `python:3.9-slim`
   - Description: The starting container image for your build
   - Required for: Container, Buildx
   - Not used for: Kaniko (Kaniko requires Dockerfile content instead), Paketo, Packer

2. **Build Instructions** (for non-Kaniko builds)
   - Type: Textarea (one instruction per line)
   - Example:
     ```
     RUN apt-get update && apt-get install -y build-essential
     COPY . /app
     WORKDIR /app
     RUN npm install && npm run build
     ```
   - Description: Dockerfile RUN, COPY, WORKDIR, EXPOSE, CMD, etc. instructions
   - Stored as: Array of strings (split by newline)
   - Required for: Container, Buildx
   - Not used for: Kaniko

3. **Tags**
   - Type: Text input (comma-separated)
   - Example: `latest, v1.0.0, production`
   - Description: Image tags to apply after build
   - Stored as: Array of strings (split by comma, trimmed)
   - Required for: All build methods
   - Used when: Pushing image to registry

**Use Cases**:
- Quick/standard container builds using Docker
- Setting up basic build instructions
- Adding image tags for versioning

**Validation**:
```
IF buildMethod != 'kaniko' AND !baseImage
  → ERROR: "Base image is required"

IF buildMethod != 'kaniko' AND !instructions.length
  → ERROR: "At least one build instruction is required"
```

---

### Tab 2: Environment

**Purpose**: Define environment variables that will be available during the build process

**Features**:
- **Dynamic key-value editor**: Add/remove environment variable pairs
- **Add button**: `+ Add Environment Variable`
- **Remove button**: `✕` per variable

**Structure**:
```javascript
environment: {
  'NPM_ENV': 'production',
  'NODE_VERSION': '18',
  'BUILD_DATE': '2026-02-01',
  'CACHE_DIR': '/tmp/cache'
}
```

**Common Use Cases**:
- `NPM_ENV=production` - Node.js environment flag
- `BUILD_FLAGS=-O3` - Compiler optimization flags
- `REGISTRY_URL=myregistry.com` - Registry endpoint
- `PYTHON_VERSION=3.9` - Language version specification
- `CI_BUILD_ID=123456` - Build metadata

**Implementation Details**:
- Stored as key-value object in `buildConfig.environment`
- UI allows inline editing of key AND value
- Empty strings are filtered on update
- Applied to the build process via build-time environment

**Applies To**: All build methods

**When Used**:
- Build-time configuration that doesn't need Dockerfile modification
- Passing secrets/credentials (though secrets manager is preferred)
- Compiler flags, language versions, feature flags

---

### Tab 3: Advanced

**Purpose**: Build-method-specific advanced configuration options

#### For Container Builds:
```tsx
Fields:
- Dockerfile Path: Default "Dockerfile"
- Build Context: Default "." (current directory)
- Target Stage: Optional (for multi-stage builds)
```

Example:
```dockerfile
# Multi-stage Dockerfile
FROM node:18 AS builder
RUN npm install && npm run build

FROM node:18-slim
COPY --from=builder /app/dist /app
```
In this case: `Target Stage = builder` selects which stage to use as output

#### For Buildx Builds:
```tsx
Fields:
- Target Platforms: e.g., "linux/amd64, linux/arm64, linux/arm/v7"
- Enable Build Cache: Checkbox
- Cache Repository: e.g., "myregistry.com/myapp/cache"
```

Example:
```
Platforms: linux/amd64, linux/arm64
Enable Cache: ✓
Cache Repo: gcr.io/my-project/build-cache
```

#### For All Build Methods:
```tsx
Fields:
- Cache Repository: Generic caching configuration
```

**Why This Tab**?
- Advanced options are method-specific and not commonly needed
- Separates "I just want to build" (Basic) from "I need advanced control" (Advanced)
- Keeps the UI clean and progressive disclosure

**Common Scenarios**:
- Multi-stage Docker builds with specific target stages
- Cross-platform builds (e.g., supporting ARM + AMD64)
- Cache optimization for large builds
- Custom Dockerfile locations (non-standard paths)

---

### Tab 4: Build Method

**Purpose**: Method-specific REQUIRED configuration that cannot be collected in other tabs

**This tab contains the unique, essential configuration for each build method:**

#### For Kaniko:
```
REQUIRED Fields:
1. Dockerfile * (Textarea - full Dockerfile content)
   - Placeholder: Complete Dockerfile example
   - Size: 8 rows
   - Validation: Cannot be empty

2. Build Context * (Text input)
   - Example: "."
   - Description: The directory with build files
   - Validation: Cannot be empty

3. Registry Repository * (Text input)
  - Example: "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app"
  - Description: Target image repository used by the Kaniko executor
  - Validation: Cannot be empty
```

**Why Kaniko is different**:
- Kaniko requires the **complete Dockerfile content as a string** (not base image + instructions)
- This is because Kaniko is Google's tool that reads an entire Dockerfile and builds without Docker daemon
- It needs the exact Dockerfile to execute

**Kaniko Validation Errors**:
```
IF !dockerfile:
  → "Dockerfile content is required for Kaniko builds"

IF !buildContext:
  → "Build context is required for Kaniko builds"

IF !registryRepo:
  → "Registry repository is required for Kaniko builds"
```

#### For Buildx:
- Uses Base Image + Instructions from **Basic tab**
- Target Platforms from **Advanced tab**
- No additional fields in Build Method tab

#### For Container:
- Uses Base Image + Instructions from **Basic tab**
- Dockerfile Path + Target Stage from **Advanced tab**
- No additional fields in Build Method tab

#### For Paketo:
```
Fields:
- Buildpack Builder (Dropdown)
  Options:
  - paketobuildpacks/builder:base
  - paketobuildpacks/builder:full
  - paketobuildpacks/builder:tiny

Validation:
  IF !builder:
    → "Buildpack builder selection is required for Paketo builds"
```

#### For Packer:
```
Fields:
- Packer Template (Textarea - HCL/JSON)
  Size: 8 rows
  Validation: Cannot be empty

Example:
{
  "builders": [
    {
      "type": "amazon-ebs",
      "region": "us-east-1",
      "source_ami": "ami-12345678"
    }
  ],
  "provisioners": [...]
}

Validation:
  IF !packerTemplate:
    → "Packer template is required for Packer builds"
```

**Information Box** at top of tab:
```
[Build Method Specific Config]
[{buildMethod.toUpperCase()} Build Configuration]
[Description of what this build method does]
```

---

## Data Flow: How Configuration is Used

### Three Important Distinctions:

1. **Frontend Local State** (WizardState)
   - Stores user input as they fill out the form
   - Uses camelCase naming (JavaScript convention)
   - Contains both UI fields AND build config

2. **API Payload Sent to Backend** (CreateBuildRequest)
   - Snake_case naming (Go backend convention)
   - Flattens/merges config data into BuildManifest
   - Includes only what the backend needs

3. **Backend Domain Model** (BuildManifest + BuildConfig)
   - Go structs with full type safety
   - Validates and stores all configuration
   - Handles different build methods with appropriate fields

### The WizardState Object (Frontend Local State):

```typescript
wizardState = {
  // From Step 1
  currentStep: number
  buildName: string
  buildDescription: string
  buildMethod: 'kaniko' | 'buildx' | 'container' | 'paketo' | 'packer'
  selectedProject: Project
  
  // From Step 2
  selectedTools: {
    sbom?: SBOMTool        // 'syft' | 'grype' | 'trivy'
    scan?: ScanTool        // 'trivy' | 'clair' | 'grype' | 'snyk'
    registry?: RegistryType // 's3' | 'harbor' | 'quay' | 'artifactory'
    secrets?: SecretManagerType // 'vault' | 'aws_secretsmanager' | 'azure_keyvault'
  }
  
  // From Step 3 Configuration
  buildConfig: {
    // Basic tab (all methods)
    baseImage?: string              // e.g., "ubuntu:20.04"
    instructions?: string[]         // Array of RUN, COPY, etc. commands
    tags?: string[]                 // ["latest", "v1.0.0"]
    
    // Environment tab (all methods)
    environment?: Record<string, string>  // {"NPM_ENV": "production"}
    
    // Advanced tab (method-specific)
    dockerfile?: string             // Path or content depending on context
    buildContext?: string            // "." or "./src"
    target?: string                  // Multi-stage target stage
    platforms?: string[]             // ["linux/amd64", "linux/arm64"]
    cache?: boolean
    cacheRepo?: string               // "myregistry.com/cache"
    
    // Build Method tab (method-specific)
    dockerfile?: string              // FULL Dockerfile content for Kaniko
    buildContext?: string            // Build context for Kaniko
    paketoConfig?: { 
      builder: string                // "paketobuildpacks/builder:base"
    }
    packerTemplate?: string          // Full Packer HCL/JSON template
  }
}
```

---

## The API Payload: CreateBuildRequest

When the user submits the build creation wizard (Step 4), the frontend **transforms** WizardState into a CreateBuildRequest and sends it to the backend API.

### Frontend Transformation (buildService.ts):

```typescript
// What the wizard collects
const buildConfig: BuildConfig = {
  buildType: wizardState.buildMethod,  // 'kaniko', 'buildx', etc.
  sbomTool: wizardState.selectedTools.sbom!,
  scanTool: wizardState.selectedTools.scan!,
  registryType: wizardState.selectedTools.registry!,
  secretManagerType: wizardState.selectedTools.secrets!,
  ...wizardState.buildConfig  // Spread all the configuration fields
}

// Transform to API payload
const buildRequest = {
  tenantId: selectedTenantId,           // Step 1: Current tenant
  projectId: wizardState.selectedProject.id,  // Step 1: Selected project
  manifest: {
    name: wizardState.buildName,        // Build name from Step 1
    description: wizardState.buildDescription,
    type: wizardState.buildMethod,      // 'kaniko', 'buildx', etc.
    baseImage: '',                      // Will be set by buildConfig
    instructions: [],                   // Will be set by buildConfig
    environment: {},                    // Will be set by buildConfig
    tags: [],                           // Will be set by buildConfig
    metadata: {},
    buildConfig                         // All the configuration!
  }
}

// Send to API
await buildService.createBuild(buildRequest)
```

### The buildService transforms to snake_case:

```typescript
async createBuild(buildData: CreateBuildRequest): Promise<Build> {
  const transformedData = {
    tenant_id: buildData.tenantId,           // camelCase → snake_case
    project_id: buildData.projectId,
    manifest: buildData.manifest
  }
  const response = await api.post('/builds', transformedData)
  return response.data.data
}
```

### API Endpoint: `POST /api/v1/builds`

**Payload Structure (JSON sent to backend)**:

```json
{
  "tenant_id": "702ea47b-22ec-439c-87b3-69f5c7e4c334",
  "project_id": "81234567-890a-bcde-f012-345678901234",
  "manifest": {
    "name": "My Node App Build",
    "description": "Building Node.js application",
    "type": "kaniko",
    "base_image": "",
    "instructions": [],
    "environment": {
      "NPM_ENV": "production",
      "NODE_VERSION": "18"
    },
    "tags": ["latest", "v1.0.0"],
    "metadata": {},
    "build_config": {
      "build_type": "kaniko",
      "sbom_tool": "syft",
      "scan_tool": "trivy",
      "registry_type": "harbor",
      "secret_manager_type": "vault",
      "dockerfile": "FROM node:18-alpine\nRUN npm install\nCMD [\"npm\", \"start\"]",
      "build_context": ".",
      "build_args": {},
      "cache": false,
      "cache_repo": ""
    }
  }
}
```

---

## Backend Processing: BuildManifest + BuildConfig

Once the API receives the payload, the backend parses it into Go structs:

### CreateBuildRequest (Handler Layer):

```go
type CreateBuildRequest struct {
  TenantID  string              `json:"tenant_id" validate:"required,uuid"`
  ProjectID string              `json:"project_id" validate:"required,uuid"`
  Manifest  build.BuildManifest `json:"manifest" validate:"required"`
}
```

### BuildManifest (Domain Layer):

```go
type BuildManifest struct {
  Name         string                 `json:"name"`
  Type         BuildType              `json:"type"`              // "kaniko", "buildx", etc.
  BaseImage    string                 `json:"base_image"`
  Instructions []string               `json:"instructions"`
  Environment  map[string]string      `json:"environment"`       // From Environment tab!
  Tags         []string               `json:"tags"`              // From Basic tab!
  Metadata     map[string]interface{} `json:"metadata"`
  BuildConfig  *BuildConfig           `json:"build_config"`      // All 3 tabs + methods
}
```

### BuildConfig (Domain Layer):

```go
type BuildConfig struct {
  // Mandatory for all builds
  BuildType         BuildType              // From wizard step selection
  SBOMTool          SBOMTool               // From Tool Selection step
  ScanTool          ScanTool               // From Tool Selection step
  RegistryType      RegistryType           // From Tool Selection step
  SecretManagerType SecretManagerType      // From Tool Selection step

  // For Kaniko & Buildx (Dockerfile-based)
  Dockerfile        string                 // FULL Dockerfile content from Build Method tab
  BuildContext      string                 // "." or custom path from Build Method tab
  BuildArgs         map[string]string      // Build arguments
  Target            string                 // Multi-stage target from Advanced tab
  Cache             bool                   // Enable cache from Advanced tab
  CacheRepo         string                 // Cache repo from Advanced tab

  // For Buildx only
  Platforms         []string               // Target platforms from Advanced tab
  CacheTo           string                 // Cache export location
  CacheFrom         []string               // Cache import locations
  Secrets           map[string]string      // Build secrets

  // For Paketo (Buildpacks)
  PaketoConfig      *PaketoConfig          // Builder + buildpacks from Build Method tab
    // PaketoConfig struct:
    // {
    //   Builder: "paketobuildpacks/builder:base"    // From Build Method tab
    //   Buildpacks: []string{}
    //   Env: map[string]string                      // From Environment tab
    //   BuildArgs: map[string]string
    // }

  // For Packer (VM images)
  PackerTemplate    string                 // Full HCL/JSON from Build Method tab
  Builders          []PackerBuilder        // Parsed from template
  Provisioners      []VMProvisioner        // Parsed from template
  PostProcessors    []VMPostProcessor      // Parsed from template

  // Common
  Variables         map[string]interface{} // Build variables
}
```

---

## Validation Flow

### When does validation happen?

**Approach: Method-Specific Validation at Bottom of Step**

Validation messages appear at the **bottom of the Configuration Step** and are shown in **real-time** as user fills fields:

```typescript
// Kaniko validation
IF buildMethod === 'kaniko' && !dockerfile
  → Show red error: "Dockerfile content is required for Kaniko builds"

IF buildMethod === 'kaniko' && !buildContext
  → Show red error: "Build context is required for Kaniko builds"

// Container/Buildx validation
IF buildMethod !== 'kaniko' && !baseImage
  → Show red error: "Base image is required"

IF buildMethod !== 'kaniko' && !instructions.length
  → Show red error: "At least one build instruction is required"

// Paketo validation
IF buildMethod === 'paketo' && !paketoConfig?.builder
  → Show red error: "Buildpack builder selection is required"

// Packer validation
IF buildMethod === 'packer' && !packerTemplate
  → Show red error: "Packer template is required"
```

### Next Step Button State:

- **Enabled**: When all required fields for current build method are filled
- **Disabled**: When validation messages are present

---

## Why This Tab Structure?

### Problem Being Solved:

Building container images has **many, many options**:
- Base image
- Build instructions
- Environment variables
- Caching strategy
- Multi-platform support
- Multi-stage targeting
- Build-method-specific fields

### Solution: Progressive Disclosure

**Tab 1 (Basic)**
- Essential for most builds
- Getting started experience

**Tab 2 (Environment)**
- Common but optional
- Grouped together by category

**Tab 3 (Advanced)**
- For power users
- Method-specific optimizations
- Not needed for simple builds

**Tab 4 (Build Method)**
- Method-specific REQUIRED fields
- Validation only for these ensures correct setup
- Clear separation of concerns

### Alternative Approaches NOT Used:

❌ **Single long form**: Would be overwhelming
❌ **Conditional fields**: Would be confusing (too many if-then display rules)
✅ **Tabs**: Clear organization, progressive disclosure, scalable

---

## Common Scenarios & Tab Usage

### Scenario 1: Simple Node.js app with Buildx
```
Step 1: Select Buildx
  ↓
Step 2: Select tools (SBOM, scanning, etc.)
  ↓
Step 3: Configuration
  ├─ Basic Tab:
  │  - Base Image: node:18-alpine
  │  - Instructions: RUN npm install, RUN npm run build, COPY . /app
  │  - Tags: latest, v1.0.0
  ├─ Environment Tab:
  │  - NODE_ENV=production
  │  - NPM_REGISTRY=https://registry.npmjs.org
  ├─ Advanced Tab:
  │  - Platforms: linux/amd64, linux/arm64
  │  - Cache: ✓ Enabled
  └─ Build Method Tab:
     - No additional fields
  ✓ Proceed to Step 4
```

### Scenario 2: Complex app with multi-stage Docker build
```
Step 3: Configuration
  ├─ Basic Tab:
  │  - Base Image: node:18
  │  - Instructions: (build and bundle commands)
  │  - Tags: latest
  ├─ Environment Tab:
  │  - (None)
  ├─ Advanced Tab:
  │  - Dockerfile Path: ./Dockerfile.prod
  │  - Target Stage: production  ← KEY: specify which stage
  │  - Build Context: .
  └─ Build Method Tab:
     - (Container build, no extra fields)
  ✓ Proceed to Step 4
```

### Scenario 3: Kaniko on Kubernetes
```
Step 3: Configuration
  ├─ Basic Tab:
  │  - Base Image: (IGNORED - Kaniko uses Dockerfile)
  │  - Instructions: (IGNORED - Kaniko uses Dockerfile)
  │  - Tags: latest, v1.0
  ├─ Environment Tab:
  │  - REGISTRY_URL=gcr.io/my-project
  │  - KANIKO_CACHE=true
  ├─ Advanced Tab:
  │  - (No Advanced options for Kaniko in Advanced tab)
  └─ Build Method Tab:
     - Dockerfile: (Full Dockerfile content pasted/edited here)
       FROM ubuntu:20.04
       RUN apt-get update && apt-get install -y build-essential
       COPY . /app
       RUN make build
       CMD ["./app"]
     - Build Context: .
  ✓ Proceed to Step 4
```

### Scenario 4: Paketo Buildpacks
```
Step 3: Configuration
  ├─ Basic Tab:
  │  - Base Image: (IGNORED - Paketo auto-detects)
  │  - Instructions: (IGNORED - Paketo auto-generates)
  │  - Tags: latest
  ├─ Environment Tab:
  │  - BP_NODE_VERSION=18
  ├─ Advanced Tab:
  │  - Cache Repository: gcr.io/my-project/cache
  └─ Build Method Tab:
     - Builder: paketobuildpacks/builder:base
  ✓ Proceed to Step 4
```

---

## Implementation Architecture

### Component: ConfigurationStep.tsx

**Props**:
```typescript
interface ConfigurationStepProps {
  wizardState: WizardState
  onUpdate: (updates: Partial<WizardState>) => void
}
```

**Internal State**:
```typescript
activeTab: 'basic' | 'advanced' | 'environment' | 'build'
```

**Key Functions**:
- `renderBasicConfig()` - Renders Tab 1
- `renderEnvironmentConfig()` - Renders Tab 2
- `renderAdvancedConfig()` - Renders Tab 3
- `renderBuildMethodSpecific()` - Renders Tab 4
- `updateBuildConfig(updates)` - Updates buildConfig in wizardState

**Key Features**:
- Tabs are simple buttons that change `activeTab` state
- Tab content rendered conditionally based on `activeTab`
- All updates call `onUpdate()` with partial WizardState
- Parent component (BuildCreationWizard) manages overall state
- Validation messages appear at bottom, updated in real-time

---

## How Each Tab's Data Flows to the Backend

### Important Clarification: Why All 3 Tabs Matter for Kaniko

**You asked**: "If Kaniko needs entire dockerfile, what purpose these 3 tabs serve?"

**Answer**: Even though Kaniko requires full Dockerfile content, the other tabs serve critical purposes:

#### Tab 1: Basic (IGNORED for Kaniko, STORED in BuildConfig)
```
Basic tab fields: baseImage, instructions, tags
├─ For Kaniko: NOT USED (Kaniko has everything in Dockerfile)
├─ For Buildx: USED (baseImage becomes first FROM line, instructions become RUN/COPY/etc)
├─ For Container: USED (same as Buildx)
└─ STORED IN: buildConfig.buildArgs, buildConfig.cache settings

Actually, here's the truth: Basic tab fields are NEVER sent to backend for Kaniko
because Kaniko doesn't need them - it uses the Dockerfile instead.

BUT: Tags ARE sent! (manifest.tags)
Build names and descriptions ARE sent! (manifest.name, manifest.description)
```

#### Tab 2: Environment (USED for ALL methods including Kaniko!)
```
Environment tab fields: key=value pairs
├─ For Kaniko: SENT as manifest.environment
│  └─ Used as build-time environment variables when Kaniko runs
│  └─ Example: NPM_ENV=production passed to the Kaniko builder
│
├─ For Buildx: SENT as manifest.environment
│  └─ Available to all RUN instructions in Dockerfile
│  └─ Example: RUN npm run build (with NPM_ENV set)
│
└─ For ALL methods:
   └─ BUILD-TIME variables, not just "nice to have"
   └─ Critical for passing config: REGISTRY_URL, BUILD_FLAGS, VERSION, etc.
```

**Real Kaniko Example**:
```dockerfile
# Dockerfile content (from Build Method tab)
FROM ubuntu:20.04

# This environment variable comes from Environment tab!
RUN echo "Building in environment: ${BUILD_ENV}"
RUN npm install
RUN npm run build --prod

ENV APP_ENV=$BUILD_ENV
```

When Kaniko builds:
```
API Payload environment: { "BUILD_ENV": "production", "NPM_REGISTRY": "https://..." }
                                        ↓ (These are available to Dockerfile)
Kaniko executes: podman build with ENV vars set
                 → RUN npm install (with NPM_REGISTRY available)
                 → RUN npm run build --prod
                 → ENV APP_ENV=production
```

#### Tab 3: Advanced (Mostly IGNORED for Kaniko, Used for other methods)
```
Advanced tab fields: dockerfile path, build context, target stage, platforms, cache
├─ For Kaniko:
│  └─ Dockerfile path: IGNORED (Kaniko uses full content from Build Method tab)
│  └─ Build context: USED! (buildConfig.buildContext = ".")
│  └─ Target stage: IGNORED (Kaniko uses Dockerfile as-is)
│  └─ Platforms: N/A
│  └─ Cache settings: STORED (buildConfig.cache, buildConfig.cacheRepo)
│
├─ For Buildx:
│  └─ Dockerfile path: USED
│  └─ Build context: USED
│  └─ Target stage: USED (multi-stage builds)
│  └─ Platforms: USED! (buildConfig.platforms = ["linux/amd64", "linux/arm64"])
│  └─ Cache: USED (buildConfig.cache, buildConfig.cacheRepo)
│
└─ For Container:
   └─ Similar to Buildx but no multi-platform support
```

#### Tab 4: Build Method (REQUIRED for each method)
```
Build Method tab fields: Method-specific configuration
├─ For Kaniko:
│  ├─ Dockerfile (REQUIRED): Full content
│  │  └─ Sent as: buildConfig.dockerfile
│  └─ Build context (REQUIRED): ".", "./src", etc.
│     └─ Sent as: buildConfig.buildContext
│
├─ For Buildx:
│  └─ No extra fields! Uses Basic + Advanced
│
├─ For Container:
│  └─ No extra fields! Uses Basic + Advanced
│
├─ For Paketo:
│  └─ Builder selection (REQUIRED)
│     └─ Sent as: buildConfig.paketoConfig.builder
│
└─ For Packer:
   └─ Packer template (REQUIRED)
      └─ Sent as: buildConfig.packerTemplate
```

---

## Complete Data Flow Example: Kaniko Build

### What User Enters:

**Step 1: Build Method** (User selects "Kaniko")
```
Project: MyProject
Build Name: My App Build
Build Method: Kaniko
```

**Step 2: Tool Selection**
```
SBOM Tool: syft
Scan Tool: trivy
Registry: harbor
Secrets Manager: vault
```

**Step 3: Configuration**

*Tab 1: Basic* (irrelevant for Kaniko, but form shows it)
```
Base Image: [ignored for Kaniko]
Instructions: [ignored for Kaniko]
Tags: latest, v1.0.0
```

*Tab 2: Environment*
```
NPM_ENV = production
NPM_REGISTRY = https://registry.npmjs.org
CACHE_DIR = /tmp/cache
```

*Tab 3: Advanced*
```
Dockerfile Path: [ignored]
Build Context: .
Cache Enabled: true
Cache Repo: harbor.mycompany.com/cache
```

*Tab 4: Build Method*
```
Dockerfile:
FROM node:18-alpine
RUN npm config set registry $NPM_REGISTRY
WORKDIR /app
COPY . .
RUN npm ci
RUN npm run build
EXPOSE 3000
CMD ["npm", "start"]

Build Context: .
```

### API Payload Sent:

```json
{
  "tenant_id": "702ea47b-22ec-439c-87b3-69f5c7e4c334",
  "project_id": "81234567-890a-bcde-f012-345678901234",
  "manifest": {
    "name": "My App Build",
    "description": "",
    "type": "kaniko",
    "base_image": "",              // ← Ignored for Kaniko
    "instructions": [],             // ← Ignored for Kaniko
    "environment": {                // ← FROM Tab 2!
      "NPM_ENV": "production",
      "NPM_REGISTRY": "https://registry.npmjs.org",
      "CACHE_DIR": "/tmp/cache"
    },
    "tags": ["latest", "v1.0.0"],   // ← FROM Tab 1!
    "metadata": {},
    "build_config": {
      "build_type": "kaniko",
      "sbom_tool": "syft",
      "scan_tool": "trivy",
      "registry_type": "harbor",
      "secret_manager_type": "vault",
      "dockerfile": "FROM node:18-alpine\nRUN npm config set registry $NPM_REGISTRY\n...",  // ← FROM Tab 4!
      "build_context": ".",         // ← FROM Tab 3!
      "build_args": {},
      "cache": true,                // ← FROM Tab 3!
      "cache_repo": "harbor.mycompany.com/cache"  // ← FROM Tab 3!
    }
  }
}
```

### How Backend Uses This:

```go
// Backend receives and parses into Go structs
func (h *BuildHandler) CreateBuild(w http.ResponseWriter, r *http.Request) {
  var req CreateBuildRequest
  json.NewDecoder(r.Body).Decode(&req)
  
  // req.Manifest contains all configuration
  // req.Manifest.BuildConfig contains the build method specific config
  
  // When Kaniko build runs:
  // 1. Start Kaniko executor with:
  //    - Dockerfile content: req.Manifest.BuildConfig.Dockerfile
  //    - Build context: req.Manifest.BuildConfig.BuildContext = "."
  //    - Environment: req.Manifest.Environment (NPM_ENV, NPM_REGISTRY, CACHE_DIR)
  //    - Cache config: req.Manifest.BuildConfig.Cache = true
  //
  // 2. After build completes:
  //    - Tag image with: req.Manifest.Tags (["latest", "v1.0.0"])
  //    - Push to: req.Manifest.BuildConfig.RegistryType (harbor)
  //    - Generate SBOM with: req.Manifest.BuildConfig.SBOMTool (syft)
  //    - Scan image with: req.Manifest.BuildConfig.ScanTool (trivy)
}
```

---

## Summary: Tab Purposes for Kaniko

| Tab | Fields | Purpose for Kaniko | Status |
|-----|--------|-------------------|--------|
| **Basic** | Base image, instructions | Ignored (Kaniko has these in Dockerfile) | ✗ Not Used |
| | Tags | Specify image tags after build | ✓ **USED** |
| **Environment** | Key-value variables | Available to Dockerfile RUN commands | ✓ **USED** |
| | (e.g., NPM_ENV, BUILD_FLAGS) | Build-time configuration | ✓ **CRITICAL** |
| **Advanced** | Dockerfile path | Not needed (using full content) | ✗ Not Used |
| | Build context | Directory with build context files (source code) | ✓ **USED** |
| | Cache settings | Enable/configure build cache | ✓ **USED** |
| **Build Method** | Dockerfile content | Full Dockerfile to execute | ✓ **REQUIRED** |
| | Build context path | Where source files are located | ✓ **REQUIRED** |



### Root Cause:
You selected **Kaniko** as your build method, which has different requirements than other methods.

### Solution:
1. Navigate to the **Build Method** tab (rightmost tab)
2. Fill in the **Dockerfile** textarea with your complete Dockerfile content
3. Fill in the **Build Context** field (usually `.` for current directory)
4. Errors should disappear

### Why Kaniko is Different:
- Kaniko is Google's container build tool
- It requires the **entire Dockerfile content as a string**, not individual instructions
- This is different from `docker build` which reads Dockerfile from filesystem
- Makes sense: Kaniko can run in restricted environments (Kubernetes pods) without Docker daemon

---

## UX Design Principles

### 1. Progressive Disclosure
- Basic users only need Tab 1
- Advanced users can explore Tabs 3 & 4
- Encourages learning curve progression

### 2. Method-Specific Guidance
- Each build method has a **blue info box** explaining its purpose
- Users know what they're building with

### 3. Real-Time Validation
- Errors appear at bottom of form
- Users see what's missing immediately
- No surprises on next step

### 4. Clear Labeling
- Each field has description text
- Placeholders show expected format
- Required fields marked with `*`

### 5. Organized Categories
- **Basic**: Essential, common to all
- **Environment**: Optional, for all
- **Advanced**: Optional, method-specific
- **Build Method**: Required, method-specific

---

## Summary Table

| Tab | Purpose | Required? | Used For | Who Fills It? |
|-----|---------|-----------|----------|--------------|
| **Basic** | Common build config | Yes | Base image, instructions, tags | All users |
| **Environment** | Build-time variables | No | ENV vars, flags, metadata | Power users |
| **Advanced** | Advanced options | No | Multi-stage, multi-platform, caching | Advanced users |
| **Build Method** | Method-specific config | Yes | Kaniko Dockerfile, Paketo builder, etc. | All users (specific to chosen method) |

---

## Next Steps

After Configuration Step:
→ **Step 4: Validation & Submission**
- Review all collected data
- Show summary of build configuration
- Final validation before creating build
- Submit to backend API
