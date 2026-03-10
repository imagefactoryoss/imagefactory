import BuildCreationWizard from '@/components/builds/BuildCreationWizard'
import React from 'react'

const CreateBuildPage: React.FC = () => {
    return (
        <div className="max-w-7xl mx-auto px-4 py-6 sm:px-6 lg:px-8">
            <BuildCreationWizard />
        </div>
    )
}

export default CreateBuildPage