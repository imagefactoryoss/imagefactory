import { BuildMethodInfo } from '../types/buildConfig';

export const BUILD_METHODS: Record<string, BuildMethodInfo> = {
    packer: {
        id: 'packer',
        name: 'Packer',
        description: 'HashiCorp Packer - Multi-cloud infrastructure as code',
        icon: '📦',
        requirements: [
            'Packer binary installed',
            'Cloud provider credentials (AWS, Azure, GCP, etc.)',
            'Packer template (HCL or JSON)',
        ],
        advantages: [
            'Multi-cloud support',
            'Infrastructure as code',
            'Parallel builds',
            'Post-processor support',
        ],
        bestFor: 'Building AMIs, Azure images, GCP images across multiple cloud providers',
        documentationUrl: 'https://www.packer.io/docs',
    },
    buildx: {
        id: 'buildx',
        name: 'Docker Buildx',
        description: 'Docker Buildx - Multi-platform Docker image builder',
        icon: '🐳',
        requirements: [
            'Docker with buildx plugin enabled',
            'Dockerfile',
            'Build context directory',
            'Docker Hub or registry credentials',
        ],
        advantages: [
            'Multi-platform builds (ARM64, x86, etc.)',
            'Native caching support',
            'BuildKit improvements',
            'OCI image support',
        ],
        bestFor: 'Building Docker images for multiple CPU architectures',
        documentationUrl: 'https://docs.docker.com/build/buildx/',
    },
    kaniko: {
        id: 'kaniko',
        name: 'Kaniko',
        description: 'Kaniko - Container image build from source in Kubernetes',
        icon: '🛢️',
        requirements: [
            'Dockerfile',
            'Container registry (ECR, GCR, Docker Hub, etc.)',
            'Registry credentials/auth',
            'Build context (git, S3, or local)',
        ],
        advantages: [
            'Rootless container builds',
            'Kubernetes-native',
            'Efficient layer caching',
            'Reproducible builds',
        ],
        bestFor: 'Building containers in Kubernetes environments without Docker daemon',
        documentationUrl: 'https://github.com/GoogleContainerTools/kaniko',
    },
    docker: {
        id: 'docker',
        name: 'Docker',
        description: 'Standard Docker image builder',
        icon: '🔧',
        requirements: [
            'Docker daemon',
            'Dockerfile',
            'Build context',
        ],
        advantages: [
            'Simple and familiar',
            'Wide ecosystem support',
            'Layer caching',
        ],
        bestFor: 'Standard Docker image builds',
        documentationUrl: 'https://docs.docker.com/build/',
    },
    paketo: {
        id: 'paketo',
        name: 'Paketo Buildpacks',
        description: 'Cloud Native Buildpacks via Paketo builders',
        icon: '🏗️',
        requirements: [
            'pack CLI installed',
            'Application source directory',
            'A compatible builder image',
        ],
        advantages: [
            'No Dockerfile required',
            'Auto-detect buildpacks',
            'Reproducible supply chain metadata',
            'Best-practice runtime images',
        ],
        bestFor: 'Application builds without writing Dockerfiles',
        documentationUrl: 'https://paketo.io/docs/',
    },
    nix: {
        id: 'nix',
        name: 'Nix',
        description: 'Nix language functional package manager',
        icon: '❄️',
        requirements: [
            'Nix installed',
            'Nix expressions (.nix files)',
            'Flake support (optional)',
        ],
        advantages: [
            'Reproducible builds',
            'Declarative configuration',
            'Powerful dependency management',
            'Multiple outputs',
        ],
        bestFor: 'Reproducible builds in various environments, complex dependencies',
        documentationUrl: 'https://nixos.org/manual/nix/stable/',
    },
};

export const BUILDX_COMMON_PLATFORMS = [
    'linux/amd64',
    'linux/arm64',
    'linux/arm/v7',
    'linux/386',
    'linux/ppc64le',
    'linux/s390x',
];

export const PACKER_ON_ERROR_MODES = [
    { value: 'ask', label: 'Ask (pause and ask)' },
    { value: 'cleanup', label: 'Cleanup (delete and abort)' },
    { value: 'abort', label: 'Abort (stop immediately)' },
];
