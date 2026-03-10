import CreateProjectModal from '@/pages/projects/CreateProjectModal'
import React from 'react'
import { useNavigate } from 'react-router-dom'

const CreateProjectPage: React.FC = () => {
    const navigate = useNavigate()

    return (
        <CreateProjectModal
            isOpen
            onClose={() => navigate('/projects')}
            onCreated={(project) => navigate(`/projects/${project.id}`)}
        />
    )
}

export default CreateProjectPage
