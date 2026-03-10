import { useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

const EditProjectPage: React.FC = () => {
    const navigate = useNavigate()
    const { projectId } = useParams<{ projectId: string }>()

    useEffect(() => {
        if (!projectId) {
            navigate('/projects', { replace: true })
            return
        }

        navigate(`/projects/${projectId}?edit=1`, { replace: true })
    }, [navigate, projectId])

    return null
}

export default EditProjectPage
