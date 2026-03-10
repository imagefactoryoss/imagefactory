import React from 'react'
import { Link } from 'react-router-dom'

const NotFoundPage: React.FC = () => {
    return (
        <div className="min-h-screen flex items-center justify-center bg-background">
            <div className="text-center">
                <h1 className="text-4xl font-bold text-primary mb-4">404</h1>
                <p className="text-xl text-muted-foreground mb-8">Page not found</p>
                <Link to="/dashboard" className="btn btn-primary">
                    Go to Dashboard
                </Link>
            </div>
        </div>
    )
}

export default NotFoundPage