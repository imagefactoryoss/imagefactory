import { InfrastructureProviderType } from '@/types'
import { ProviderFormComponent } from './types'

// Lazy load provider components
const providerComponents: Record<InfrastructureProviderType, () => Promise<ProviderFormComponent>> = {
    kubernetes: () => import('./StandardKubernetesForm').then(m => m.StandardKubernetesForm),
    'aws-eks': () => import('./AWSEKSForm').then(m => m.AWSEKSForm),
    'gcp-gke': () => import('./GCPGKEForm').then(m => m.GCPGKEForm),
    'azure-aks': () => import('./AzureAKSForm').then(m => m.AzureAKSForm),
    'oci-oke': () => import('./OCIOKEForm').then(m => m.OCIOKEForm),
    'vmware-vks': () => import('./VMwareVKSForm').then(m => m.VMwareVKSForm),
    'openshift': () => import('./OpenShiftForm').then(m => m.OpenShiftForm),
    'rancher': () => import('./RancherForm').then(m => m.RancherForm),
    'build_nodes': () => import('./BuildNodesForm').then(m => m.BuildNodesForm),
}

class ProviderRegistry {
    private loadedComponents = new Map<InfrastructureProviderType, ProviderFormComponent>()

    async getProviderForm(providerType: InfrastructureProviderType): Promise<ProviderFormComponent> {
        // Check if already loaded
        if (this.loadedComponents.has(providerType)) {
            return this.loadedComponents.get(providerType)!
        }

        // Load the component
        const loader = providerComponents[providerType]
        if (!loader) {
            throw new Error(`Unknown provider type: ${providerType}`)
        }

        const component = await loader()
        this.loadedComponents.set(providerType, component)
        return component
    }

    getAllProviderTypes(): InfrastructureProviderType[] {
        return Object.keys(providerComponents) as InfrastructureProviderType[]
    }

    isProviderTypeSupported(providerType: string): providerType is InfrastructureProviderType {
        return providerType in providerComponents
    }
}

export const providerRegistry = new ProviderRegistry()