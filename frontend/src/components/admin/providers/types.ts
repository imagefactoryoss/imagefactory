import { InfrastructureProviderType } from '@/types'
import React from 'react'

export interface ProviderFormData {
    provider_type: InfrastructureProviderType
    name: string
    display_name: string
    config: Record<string, any>
    capabilities: string[]
    target_namespace?: string
    bootstrap_mode?: 'image_factory_managed' | 'self_managed' | string
}

export interface ProviderConfigFormProps {
    formData: ProviderFormData
    setFormData: (data: ProviderFormData) => void
    isEditing?: boolean
}

export interface ProviderFormComponent {
    component: React.ComponentType<ProviderConfigFormProps>
    displayName: string
    description: string
    icon?: React.ReactNode
}
