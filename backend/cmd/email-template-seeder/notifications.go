package main

// getNotificationTemplates returns notification related templates
func getNotificationTemplates() []EmailTemplate {
	templates := []EmailTemplate{
		getBuildStartedTemplate(),
		getBuildCompletedTemplate(),
		getBuildFailedTemplate(),
		getBuildCancelledTemplate(),
		getDeploymentStartedTemplate(),
		getDeploymentCompletedTemplate(),
		getDeploymentFailedTemplate(),
		getImageReadyTemplate(),
		getImageDeprecatedTemplate(),
		getImageSecurityAlertTemplate(),
		getImagePublishedTemplate(),
	}

	templates = append(templates,
		getWorkflowEventTemplate("external_image_import_approval_requested", "Quarantine Approval Requested"),
		getWorkflowEventTemplate("external_image_import_approved", "Quarantine Request Approved"),
		getWorkflowEventTemplate("external_image_import_rejected", "Quarantine Request Rejected"),
		getWorkflowEventTemplate("external_image_import_dispatch_failed", "Quarantine Import Dispatch Failed"),
		getWorkflowEventTemplate("external_image_import_completed", "Quarantine Import Completed"),
		getWorkflowEventTemplate("external_image_import_quarantined", "Image Quarantined"),
		getWorkflowEventTemplate("external_image_import_failed", "Quarantine Import Failed"),
		getWorkflowEventTemplate("epr_registration_requested", "EPR Registration Requested"),
		getWorkflowEventTemplate("epr_registration_approved", "EPR Registration Approved"),
		getWorkflowEventTemplate("epr_registration_rejected", "EPR Registration Rejected"),
		getWorkflowEventTemplate("epr_registration_suspended", "EPR Registration Suspended"),
		getWorkflowEventTemplate("epr_registration_reactivated", "EPR Registration Reactivated"),
		getWorkflowEventTemplate("epr_registration_revalidated", "EPR Registration Revalidated"),
		getWorkflowEventTemplate("epr_registration_expiring", "EPR Registration Expiring"),
		getWorkflowEventTemplate("epr_registration_expired", "EPR Registration Expired"),
	)

	return templates
}

func getWorkflowEventTemplate(templateType, title string) EmailTemplate {
	return EmailTemplate{
		TemplateType: templateType,
		Category:     CategoryNotifications,
		Subject:      "[Image Factory] {{.NotificationTitle}}",
		Description:  "Workflow notification for " + templateType,
		TextTemplate: `{{.NotificationTitle}}

Hello {{.UserName}},

{{.Message}}

Request Details
---------------
Status: {{.Status}}
Request Type: {{.RequestType}}
Source Image: {{.SourceImageRef}}
{{if .EPRRecordID}}EPR Record: {{.EPRRecordID}}{{end}}
{{if .ProductName}}Product: {{.ProductName}}{{end}}
{{if .TechnologyName}}Technology: {{.TechnologyName}}{{end}}

{{if .DashboardURL}}Dashboard: {{.DashboardURL}}{{end}}

Regards,
Image Factory Platform`,
		HTMLTemplate: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.NotificationTitle}}</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; line-height: 1.6; color: #1f2937; margin: 0; padding: 0; background-color: #f3f4f6; }
        .container { max-width: 640px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 12px rgba(15, 23, 42, 0.08); }
        .header { background: linear-gradient(135deg, #1d4ed8 0%, #0f172a 100%); color: #ffffff; padding: 28px 24px; }
        .header h1 { margin: 0; font-size: 22px; font-weight: 600; }
        .header p { margin: 6px 0 0 0; opacity: 0.9; font-size: 13px; }
        .content { padding: 24px; }
        .message { margin: 0 0 16px 0; }
        .details { border: 1px solid #e5e7eb; background-color: #f9fafb; border-radius: 10px; padding: 14px; margin-top: 12px; }
        .details p { margin: 6px 0; font-size: 14px; }
        .label { color: #6b7280; font-weight: 600; display: inline-block; min-width: 120px; }
        .cta { margin-top: 18px; }
        .cta a { display: inline-block; background: #2563eb; color: #ffffff; text-decoration: none; padding: 10px 16px; border-radius: 8px; font-weight: 600; font-size: 14px; }
        .footer { padding: 16px 24px; font-size: 12px; color: #6b7280; border-top: 1px solid #e5e7eb; background-color: #fafafa; }
    </style>
</head>
<body>
    <div style="padding:24px 16px;">
        <div class="container">
            <div class="header">
                <h1>{{.NotificationTitle}}</h1>
                <p>Image Factory Notification</p>
            </div>
            <div class="content">
                <p class="message">Hello {{.UserName}},</p>
                <p class="message">{{.Message}}</p>
                <div class="details">
                    <p><span class="label">Status</span>{{.Status}}</p>
                    <p><span class="label">Request Type</span>{{.RequestType}}</p>
                    <p><span class="label">Source Image</span>{{.SourceImageRef}}</p>
                    {{if .EPRRecordID}}<p><span class="label">EPR Record</span>{{.EPRRecordID}}</p>{{end}}
                    {{if .ProductName}}<p><span class="label">Product</span>{{.ProductName}}</p>{{end}}
                    {{if .TechnologyName}}<p><span class="label">Technology</span>{{.TechnologyName}}</p>{{end}}
                </div>
                {{if .DashboardURL}}
                <div class="cta">
                    <a href="{{.DashboardURL}}">Open Dashboard</a>
                </div>
                {{end}}
            </div>
            <div class="footer">
                This is an automated message from Image Factory Platform.
            </div>
        </div>
    </div>
</body>
</html>`,
	}
}

// getBuildCancelledTemplate returns the build cancelled notification template
func getBuildCancelledTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: "build_cancelled",
		Category:     CategoryNotifications,
		Subject:      "Build cancelled for {{.ProjectName}} - {{.BuildID}}",
		Description:  "Notification when a build is cancelled",
		TextTemplate: `Build Cancelled

Hello {{.UserName}},

The build has been cancelled.

BUILD INFORMATION
=================
• Project: {{.ProjectName}}
• Build ID: {{.BuildID}}
• Cancelled At: {{.FailureTime}}
• Reason: {{.ErrorMessage}}

VIEW DETAILS
============
Dashboard: {{.DashboardURL}}
Build Logs: {{.BuildLogsURL}}
`,
		HTMLTemplate: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Build Cancelled</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; background-color: #f8fafc; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); }
        .header { background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%); padding: 40px 30px; text-align: center; color: white; }
        .content { padding: 32px 30px; }
        .info { background: #fff7ed; border-left: 4px solid #f59e0b; padding: 16px; border-radius: 8px; }
        .cta { margin-top: 20px; }
        .cta a { display: inline-block; background: #f59e0b; color: #fff; text-decoration: none; padding: 10px 16px; border-radius: 6px; margin-right: 8px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Build Cancelled</h1>
            <p>{{.ProjectName}} - {{.BuildID}}</p>
        </div>
        <div class="content">
            <p>Hello {{.UserName}},</p>
            <p>Your build was cancelled.</p>
            <div class="info">
                <p><strong>Build ID:</strong> {{.BuildID}}</p>
                <p><strong>Cancelled At:</strong> {{.FailureTime}}</p>
                <p><strong>Reason:</strong> {{.ErrorMessage}}</p>
            </div>
            <div class="cta">
                <a href="{{.DashboardURL}}">View Dashboard</a>
                <a href="{{.BuildLogsURL}}">View Build Logs</a>
            </div>
        </div>
    </div>
</body>
</html>`,
	}
}

// getBuildStartedTemplate returns the build started notification template
func getBuildStartedTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: "build_started",
		Category:     CategoryNotifications,
		Subject:      "Build started for {{.ProjectName}} - {{.BuildID}}",
		Description:  "Notification when a new build is started",
		TextTemplate: `Build Started

Hello {{.UserName}},

A new build has been started for your project.

BUILD INFORMATION
=================
• Project: {{.ProjectName}}
• Build ID: {{.BuildID}}
• Branch: {{.Branch}}
• Commit: {{.CommitHash}}
• Started At: {{.StartTime}}
• Triggered By: {{.TriggeredBy}}

VIEW BUILD DETAILS
==================
Dashboard: {{.DashboardURL}}
Build Logs: {{.BuildLogsURL}}

You will receive another notification when the build completes.

Best regards,
The Image Factory Team

---
Image Factory - Secure Container Platform for Modern Development`,
		HTMLTemplate: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Build Started</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; background-color: #f8fafc; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); }
        .header { background: linear-gradient(135deg, #3b82f6 0%, #1e40af 100%); padding: 40px 30px; text-align: center; color: white; }
        .header h1 { margin: 0; font-size: 28px; font-weight: 600; }
        .header p { margin: 8px 0 0 0; opacity: 0.9; font-size: 14px; }
        .content { padding: 40px 30px; }
        .status-badge { display: inline-block; background-color: #dbeafe; color: #1e40af; padding: 8px 16px; border-radius: 20px; font-weight: 600; font-size: 14px; margin-bottom: 20px; }
        .info-section { background-color: #f9fafb; border-radius: 8px; padding: 24px; margin: 24px 0; border-left: 4px solid #3b82f6; }
        .info-section h3 { margin: 0 0 16px 0; color: #1f2937; font-size: 18px; font-weight: 600; }
        .info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
        .info-item { display: flex; flex-direction: column; }
        .info-label { font-weight: 600; color: #6b7280; font-size: 12px; text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 4px; }
        .info-value { color: #1f2937; font-size: 14px; font-weight: 500; }
        .cta-section { text-align: center; margin: 32px 0; }
        .cta-button { display: inline-block; background: #3b82f6; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 14px; margin: 8px; }
        .footer { background-color: #1f2937; color: #9ca3af; padding: 32px 30px; text-align: center; font-size: 12px; }
        .footer a { color: #667eea; text-decoration: none; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Build Started</h1>
            <p>{{.ProjectName}} - {{.BuildID}}</p>
        </div>
        <div class="content">
            <span class="status-badge">IN PROGRESS</span>
            <p>Hello {{.UserName}},</p>
            <p>A new build has been started for your project. The build process is now running.</p>
            <div class="info-section">
                <h3>Build Details</h3>
                <div class="info-grid">
                    <div class="info-item"><span class="info-label">Build ID</span><span class="info-value">{{.BuildID}}</span></div>
                    <div class="info-item"><span class="info-label">Branch</span><span class="info-value">{{.Branch}}</span></div>
                    <div class="info-item"><span class="info-label">Commit</span><span class="info-value">{{.CommitHash}}</span></div>
                    <div class="info-item"><span class="info-label">Triggered By</span><span class="info-value">{{.TriggeredBy}}</span></div>
                    <div class="info-item"><span class="info-label">Started At</span><span class="info-value">{{.StartTime}}</span></div>
                </div>
            </div>
            <div class="cta-section">
                <a href="{{.DashboardURL}}" class="cta-button">View Dashboard</a>
                <a href="{{.BuildLogsURL}}" class="cta-button">View Build Logs</a>
            </div>
        </div>
        <div class="footer">
            <p>Image Factory - Secure Container Platform for Modern Development</p>
            <p><a href="mailto:support@imgfactory.com">support@imgfactory.com</a></p>
        </div>
    </div>
</body>
</html>`,
	}
}

// getBuildCompletedTemplate returns the build completed notification template
func getBuildCompletedTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: "build_completed",
		Category:     CategoryNotifications,
		Subject:      "Build completed for {{.ProjectName}} - {{.BuildID}}",
		Description:  "Notification when a build completes successfully",
		TextTemplate: `Build Completed Successfully

Hello {{.UserName}},

Your build has completed successfully!

BUILD SUMMARY
=============
• Project: {{.ProjectName}}
• Build ID: {{.BuildID}}
• Status: SUCCESS
• Duration: {{.Duration}}
• Image: {{.ImageName}}:{{.ImageTag}}

BUILD ARTIFACTS
===============
• Image Size: {{.ImageSize}}
• Layers: {{.LayerCount}}
• Registry: {{.RegistryURL}}

NEXT STEPS
==========
Your image is ready for deployment. You can now:
1. Deploy to your environments
2. Run tests on the image
3. Push to your registry

VIEW DETAILS
============
Dashboard: {{.DashboardURL}}
Build Logs: {{.BuildLogsURL}}
Image Details: {{.ImageDetailsURL}}

Best regards,
The Image Factory Team

---
Image Factory - Secure Container Platform for Modern Development`,
		HTMLTemplate: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Build Completed</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; background-color: #f8fafc; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); }
        .header { background: linear-gradient(135deg, #10b981 0%, #059669 100%); padding: 40px 30px; text-align: center; color: white; }
        .header h1 { margin: 0; font-size: 28px; font-weight: 600; }
        .header p { margin: 8px 0 0 0; opacity: 0.9; font-size: 14px; }
        .content { padding: 40px 30px; }
        .status-badge { display: inline-block; background-color: #d1fae5; color: #065f46; padding: 8px 16px; border-radius: 20px; font-weight: 600; font-size: 14px; margin-bottom: 20px; }
        .info-section { background-color: #f9fafb; border-radius: 8px; padding: 24px; margin: 24px 0; border-left: 4px solid #10b981; }
        .info-section h3 { margin: 0 0 16px 0; color: #1f2937; font-size: 18px; font-weight: 600; }
        .info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
        .info-item { display: flex; flex-direction: column; }
        .info-label { font-weight: 600; color: #6b7280; font-size: 12px; text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 4px; }
        .info-value { color: #1f2937; font-size: 14px; font-weight: 500; }
        .cta-section { text-align: center; margin: 32px 0; }
        .cta-button { display: inline-block; background: #10b981; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 14px; margin: 8px; }
        .footer { background-color: #1f2937; color: #9ca3af; padding: 32px 30px; text-align: center; font-size: 12px; }
        .footer a { color: #667eea; text-decoration: none; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Build Completed</h1>
            <p>{{.ProjectName}} - {{.BuildID}}</p>
        </div>
        <div class="content">
            <span class="status-badge">SUCCESS</span>
            <p>Hello {{.UserName}},</p>
            <p>Your build has completed successfully! Your container image is ready for deployment.</p>
            <div class="info-section">
                <h3>Build Summary</h3>
                <div class="info-grid">
                    <div class="info-item"><span class="info-label">Status</span><span class="info-value">SUCCESS</span></div>
                    <div class="info-item"><span class="info-label">Duration</span><span class="info-value">{{.Duration}}</span></div>
                    <div class="info-item"><span class="info-label">Image</span><span class="info-value">{{.ImageName}}:{{.ImageTag}}</span></div>
                    <div class="info-item"><span class="info-label">Size</span><span class="info-value">{{.ImageSize}}</span></div>
                </div>
            </div>
            <div class="cta-section">
                <a href="{{.DashboardURL}}" class="cta-button">View Dashboard</a>
                <a href="{{.ImageDetailsURL}}" class="cta-button">View Image</a>
            </div>
        </div>
        <div class="footer">
            <p>Image Factory - Secure Container Platform for Modern Development</p>
            <p><a href="mailto:support@imgfactory.com">support@imgfactory.com</a></p>
        </div>
    </div>
</body>
</html>`,
	}
}

// getBuildFailedTemplate returns the build failed notification template
func getBuildFailedTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: "build_failed",
		Category:     CategoryNotifications,
		Subject:      "Build failed for {{.ProjectName}} - {{.BuildID}}",
		Description:  "Notification when a build fails",
		TextTemplate: `Build Failed

Hello {{.UserName}},

Unfortunately, your build has failed. Please review the error details below.

BUILD INFORMATION
=================
• Project: {{.ProjectName}}
• Build ID: {{.BuildID}}
• Status: FAILED
• Failed At: {{.FailureTime}}
• Duration: {{.Duration}}

ERROR DETAILS
=============
{{.ErrorMessage}}

TROUBLESHOOTING
===============
1. Check the build logs for detailed error information
2. Verify your Dockerfile and build configuration
3. Ensure all dependencies are available
4. Check for syntax errors in your code

NEXT STEPS
==========
Dashboard: {{.DashboardURL}}
Build Logs: {{.BuildLogsURL}}

Need help? Contact our support team at support@imgfactory.com

Best regards,
The Image Factory Team

---
Image Factory - Secure Container Platform for Modern Development`,
		HTMLTemplate: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Build Failed</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; background-color: #f8fafc; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); }
        .header { background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%); padding: 40px 30px; text-align: center; color: white; }
        .header h1 { margin: 0; font-size: 28px; font-weight: 600; }
        .header p { margin: 8px 0 0 0; opacity: 0.9; font-size: 14px; }
        .content { padding: 40px 30px; }
        .status-badge { display: inline-block; background-color: #fee2e2; color: #7f1d1d; padding: 8px 16px; border-radius: 20px; font-weight: 600; font-size: 14px; margin-bottom: 20px; }
        .error-section { background-color: #fef2f2; border-radius: 8px; padding: 24px; margin: 24px 0; border-left: 4px solid #ef4444; }
        .error-section h3 { margin: 0 0 16px 0; color: #7f1d1d; font-size: 18px; font-weight: 600; }
        .error-text { background-color: white; border: 1px solid #fecaca; border-radius: 6px; padding: 16px; font-family: monospace; font-size: 12px; color: #7f1d1d; white-space: pre-wrap; word-break: break-word; }
        .info-section { background-color: #f9fafb; border-radius: 8px; padding: 24px; margin: 24px 0; border-left: 4px solid #6b7280; }
        .info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
        .info-label { font-weight: 600; color: #6b7280; font-size: 12px; text-transform: uppercase; }
        .info-value { color: #1f2937; font-size: 14px; font-weight: 500; }
        .cta-section { text-align: center; margin: 32px 0; }
        .cta-button { display: inline-block; background: #ef4444; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 14px; margin: 8px; }
        .footer { background-color: #1f2937; color: #9ca3af; padding: 32px 30px; text-align: center; font-size: 12px; }
        .footer a { color: #667eea; text-decoration: none; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Build Failed</h1>
            <p>{{.ProjectName}} - {{.BuildID}}</p>
        </div>
        <div class="content">
            <span class="status-badge">FAILED</span>
            <p>Hello {{.UserName}},</p>
            <p>Unfortunately, your build has failed. Please review the error details to troubleshoot the issue.</p>
            <div class="error-section">
                <h3>Error Details</h3>
                <div class="error-text">{{.ErrorMessage}}</div>
            </div>
            <div class="info-section">
                <h3>Build Information</h3>
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px;">
                    <div><span class="info-label">Build ID</span><span class="info-value">{{.BuildID}}</span></div>
                    <div><span class="info-label">Failed At</span><span class="info-value">{{.FailureTime}}</span></div>
                    <div><span class="info-label">Duration</span><span class="info-value">{{.Duration}}</span></div>
                </div>
            </div>
            <div class="cta-section">
                <a href="{{.BuildLogsURL}}" class="cta-button">View Build Logs</a>
                <a href="{{.DashboardURL}}" class="cta-button">View Dashboard</a>
            </div>
        </div>
        <div class="footer">
            <p>Image Factory - Secure Container Platform for Modern Development</p>
            <p><a href="mailto:support@imgfactory.com">support@imgfactory.com</a></p>
        </div>
    </div>
</body>
</html>`,
	}
}

// getDeploymentStartedTemplate returns the deployment started notification template
func getDeploymentStartedTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: "deployment_started",
		Category:     CategoryNotifications,
		Subject:      "Deployment started for {{.ProjectName}} to {{.Environment}}",
		Description:  "Notification when a deployment is started",
		TextTemplate: `Deployment Started

Hello {{.UserName}},

A new deployment has been initiated for your project.

DEPLOYMENT INFORMATION
======================
• Project: {{.ProjectName}}
• Deployment ID: {{.DeploymentID}}
• Environment: {{.Environment}}
• Image: {{.ImageName}}:{{.ImageTag}}
• Initiated By: {{.InitiatedBy}}
• Started At: {{.StartTime}}

MONITORING
==========
Monitor the deployment progress: {{.DeploymentLogsURL}}
Status Page: {{.StatusPageURL}}

You will receive another notification when the deployment completes.

Best regards,
The Image Factory Team

---
Image Factory - Secure Container Platform for Modern Development`,
		HTMLTemplate: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Deployment Started</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; background-color: #f8fafc; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); }
        .header { background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%); padding: 40px 30px; text-align: center; color: white; }
        .header h1 { margin: 0; font-size: 28px; font-weight: 600; }
        .content { padding: 40px 30px; }
        .status-badge { display: inline-block; background-color: #fef3c7; color: #92400e; padding: 8px 16px; border-radius: 20px; font-weight: 600; font-size: 14px; margin-bottom: 20px; }
        .info-section { background-color: #f9fafb; border-radius: 8px; padding: 24px; margin: 24px 0; border-left: 4px solid #f59e0b; }
        .info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
        .info-label { font-weight: 600; color: #6b7280; font-size: 12px; text-transform: uppercase; margin-bottom: 4px; }
        .info-value { color: #1f2937; font-size: 14px; font-weight: 500; }
        .cta-section { text-align: center; margin: 32px 0; }
        .cta-button { display: inline-block; background: #f59e0b; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 14px; margin: 8px; }
        .footer { background-color: #1f2937; color: #9ca3af; padding: 32px 30px; text-align: center; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Deployment Started</h1>
            <p>{{.ProjectName}} to {{.Environment}}</p>
        </div>
        <div class="content">
            <span class="status-badge">IN PROGRESS</span>
            <div class="info-section">
                <h3>Deployment Details</h3>
                <div class="info-grid">
                    <div><span class="info-label">Deployment ID</span><span class="info-value">{{.DeploymentID}}</span></div>
                    <div><span class="info-label">Environment</span><span class="info-value">{{.Environment}}</span></div>
                    <div><span class="info-label">Image</span><span class="info-value">{{.ImageName}}:{{.ImageTag}}</span></div>
                    <div><span class="info-label">Initiated By</span><span class="info-value">{{.InitiatedBy}}</span></div>
                </div>
            </div>
            <div class="cta-section">
                <a href="{{.DeploymentLogsURL}}" class="cta-button">View Deployment Logs</a>
                <a href="{{.StatusPageURL}}" class="cta-button">Status Page</a>
            </div>
        </div>
        <div class="footer">
            <p>Image Factory - Secure Container Platform for Modern Development</p>
        </div>
    </div>
</body>
</html>`,
	}
}

// getDeploymentCompletedTemplate returns the deployment completed notification template
func getDeploymentCompletedTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: "deployment_completed",
		Category:     CategoryNotifications,
		Subject:      "Deployment completed for {{.ProjectName}} to {{.Environment}}",
		Description:  "Notification when a deployment completes successfully",
		TextTemplate: `Deployment Completed Successfully

Hello {{.UserName}},

Your deployment has completed successfully!

DEPLOYMENT SUMMARY
==================
• Project: {{.ProjectName}}
• Deployment ID: {{.DeploymentID}}
• Environment: {{.Environment}}
• Status: SUCCESS
• Duration: {{.Duration}}
• Image: {{.ImageName}}:{{.ImageTag}}

DEPLOYMENT DETAILS
==================
• Instances: {{.InstanceCount}}
• Region: {{.Region}}
• Replicas Healthy: {{.HealthyReplicas}}/{{.TotalReplicas}}

SERVICE URLS
============
{{.ServiceURLs}}

NEXT STEPS
==========
1. Test your application in the {{.Environment}} environment
2. Monitor application performance
3. Review deployment logs if needed

VIEW DETAILS
============
Dashboard: {{.DashboardURL}}
Logs: {{.DeploymentLogsURL}}

Best regards,
The Image Factory Team

---
Image Factory - Secure Container Platform for Modern Development`,
		HTMLTemplate: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Deployment Completed</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; background-color: #f8fafc; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); }
        .header { background: linear-gradient(135deg, #10b981 0%, #059669 100%); padding: 40px 30px; text-align: center; color: white; }
        .header h1 { margin: 0; font-size: 28px; font-weight: 600; }
        .content { padding: 40px 30px; }
        .status-badge { display: inline-block; background-color: #d1fae5; color: #065f46; padding: 8px 16px; border-radius: 20px; font-weight: 600; font-size: 14px; margin-bottom: 20px; }
        .info-section { background-color: #f9fafb; border-radius: 8px; padding: 24px; margin: 24px 0; border-left: 4px solid #10b981; }
        .info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
        .info-label { font-weight: 600; color: #6b7280; font-size: 12px; text-transform: uppercase; margin-bottom: 4px; }
        .info-value { color: #1f2937; font-size: 14px; font-weight: 500; }
        .cta-section { text-align: center; margin: 32px 0; }
        .cta-button { display: inline-block; background: #10b981; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 14px; margin: 8px; }
        .footer { background-color: #1f2937; color: #9ca3af; padding: 32px 30px; text-align: center; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Deployment Completed</h1>
            <p>{{.ProjectName}} to {{.Environment}}</p>
        </div>
        <div class="content">
            <span class="status-badge">SUCCESS</span>
            <div class="info-section">
                <h3>Deployment Summary</h3>
                <div class="info-grid">
                    <div><span class="info-label">Status</span><span class="info-value">SUCCESS</span></div>
                    <div><span class="info-label">Duration</span><span class="info-value">{{.Duration}}</span></div>
                    <div><span class="info-label">Replicas</span><span class="info-value">{{.HealthyReplicas}}/{{.TotalReplicas}}</span></div>
                    <div><span class="info-label">Region</span><span class="info-value">{{.Region}}</span></div>
                </div>
            </div>
            <div class="cta-section">
                <a href="{{.DashboardURL}}" class="cta-button">View Dashboard</a>
                <a href="{{.DeploymentLogsURL}}" class="cta-button">View Logs</a>
            </div>
        </div>
        <div class="footer">
            <p>Image Factory - Secure Container Platform for Modern Development</p>
        </div>
    </div>
</body>
</html>`,
	}
}

// getDeploymentFailedTemplate returns the deployment failed notification template
func getDeploymentFailedTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: "deployment_failed",
		Category:     CategoryNotifications,
		Subject:      "Deployment failed for {{.ProjectName}} to {{.Environment}}",
		Description:  "Notification when a deployment fails",
		TextTemplate: `Deployment Failed

Hello {{.UserName}},

Unfortunately, your deployment to {{.Environment}} has failed.

DEPLOYMENT INFORMATION
======================
• Project: {{.ProjectName}}
• Deployment ID: {{.DeploymentID}}
• Environment: {{.Environment}}
• Status: FAILED
• Failed At: {{.FailureTime}}
• Reason: {{.FailureReason}}

AFFECTED SERVICES
=================
{{.AffectedServices}}

TROUBLESHOOTING
===============
1. Check the deployment logs for detailed error information
2. Verify the image is valid and accessible
3. Check environment configuration and secrets
4. Verify resource availability (CPU, memory)
5. Review pod events for clues

NEXT STEPS
==========
Deployment Logs: {{.DeploymentLogsURL}}
Dashboard: {{.DashboardURL}}

Need help? Contact our support team at support@imgfactory.com

Best regards,
The Image Factory Team

---
Image Factory - Secure Container Platform for Modern Development`,
		HTMLTemplate: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Deployment Failed</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; background-color: #f8fafc; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); }
        .header { background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%); padding: 40px 30px; text-align: center; color: white; }
        .header h1 { margin: 0; font-size: 28px; font-weight: 600; }
        .content { padding: 40px 30px; }
        .status-badge { display: inline-block; background-color: #fee2e2; color: #7f1d1d; padding: 8px 16px; border-radius: 20px; font-weight: 600; font-size: 14px; margin-bottom: 20px; }
        .error-section { background-color: #fef2f2; border-radius: 8px; padding: 24px; margin: 24px 0; border-left: 4px solid #ef4444; }
        .error-text { background-color: white; border: 1px solid #fecaca; border-radius: 6px; padding: 16px; font-family: monospace; font-size: 12px; color: #7f1d1d; }
        .info-section { background-color: #f9fafb; border-radius: 8px; padding: 24px; margin: 24px 0; }
        .cta-section { text-align: center; margin: 32px 0; }
        .cta-button { display: inline-block; background: #ef4444; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 14px; margin: 8px; }
        .footer { background-color: #1f2937; color: #9ca3af; padding: 32px 30px; text-align: center; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Deployment Failed</h1>
            <p>{{.ProjectName}} to {{.Environment}}</p>
        </div>
        <div class="content">
            <span class="status-badge">FAILED</span>
            <div class="error-section">
                <h3>Failure Details</h3>
                <div class="error-text">{{.FailureReason}}</div>
            </div>
            <div class="info-section">
                <h3>Deployment Information</h3>
                <p><strong>Deployment ID:</strong> {{.DeploymentID}}</p>
                <p><strong>Environment:</strong> {{.Environment}}</p>
                <p><strong>Failed At:</strong> {{.FailureTime}}</p>
            </div>
            <div class="cta-section">
                <a href="{{.DeploymentLogsURL}}" class="cta-button">View Deployment Logs</a>
                <a href="{{.DashboardURL}}" class="cta-button">View Dashboard</a>
            </div>
        </div>
        <div class="footer">
            <p>Image Factory - Secure Container Platform for Modern Development</p>
            <p><a href="mailto:support@imgfactory.com">support@imgfactory.com</a></p>
        </div>
    </div>
</body>
</html>`,
	}
}

// getImageReadyTemplate returns the image ready notification template
func getImageReadyTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: "image_ready",
		Category:     CategoryNotifications,
		Subject:      "Container image ready: {{.ImageName}}:{{.ImageTag}}",
		Description:  "Notification when a container image is ready for use",
		TextTemplate: `Container Image Ready

Hello {{.UserName}},

Your container image has been built and is ready for use!

IMAGE INFORMATION
=================
• Image Name: {{.ImageName}}
• Tag: {{.ImageTag}}
• Build ID: {{.BuildID}}
• Size: {{.ImageSize}}
• Created At: {{.CreatedAt}}

PULL COMMAND
============
docker pull {{.RegistryURL}}/{{.ImageName}}:{{.ImageTag}}

IMAGE DETAILS
=============
• Base Image: {{.BaseImage}}
• Layers: {{.LayerCount}}
• Scan Status: {{.ScanStatus}}

NEXT STEPS
==========
1. Pull the image to your local environment
2. Test the image locally
3. Deploy to your staging or production environment

VIEW DETAILS
============
Dashboard: {{.DashboardURL}}
Image Details: {{.ImageDetailsURL}}

Best regards,
The Image Factory Team

---
Image Factory - Secure Container Platform for Modern Development`,
		HTMLTemplate: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Image Ready</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; background-color: #f8fafc; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); }
        .header { background: linear-gradient(135deg, #8b5cf6 0%, #6d28d9 100%); padding: 40px 30px; text-align: center; color: white; }
        .header h1 { margin: 0; font-size: 28px; font-weight: 600; }
        .content { padding: 40px 30px; }
        .status-badge { display: inline-block; background-color: #ede9fe; color: #5b21b6; padding: 8px 16px; border-radius: 20px; font-weight: 600; font-size: 14px; margin-bottom: 20px; }
        .info-section { background-color: #f9fafb; border-radius: 8px; padding: 24px; margin: 24px 0; border-left: 4px solid #8b5cf6; }
        .info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
        .info-label { font-weight: 600; color: #6b7280; font-size: 12px; text-transform: uppercase; margin-bottom: 4px; }
        .info-value { color: #1f2937; font-size: 14px; font-weight: 500; }
        .command-box { background-color: #1f2937; color: #10b981; border-radius: 6px; padding: 16px; margin: 16px 0; font-family: monospace; font-size: 13px; word-break: break-all; }
        .cta-section { text-align: center; margin: 32px 0; }
        .cta-button { display: inline-block; background: #8b5cf6; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 14px; margin: 8px; }
        .footer { background-color: #1f2937; color: #9ca3af; padding: 32px 30px; text-align: center; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Image Ready</h1>
            <p>{{.ImageName}}:{{.ImageTag}}</p>
        </div>
        <div class="content">
            <span class="status-badge">READY</span>
            <div class="info-section">
                <h3>Image Details</h3>
                <div class="info-grid">
                    <div><span class="info-label">Image Name</span><span class="info-value">{{.ImageName}}</span></div>
                    <div><span class="info-label">Tag</span><span class="info-value">{{.ImageTag}}</span></div>
                    <div><span class="info-label">Size</span><span class="info-value">{{.ImageSize}}</span></div>
                    <div><span class="info-label">Layers</span><span class="info-value">{{.LayerCount}}</span></div>
                </div>
            </div>
            <div class="info-section">
                <h3>Pull Command</h3>
                <div class="command-box">docker pull {{.RegistryURL}}/{{.ImageName}}:{{.ImageTag}}</div>
            </div>
            <div class="cta-section">
                <a href="{{.DashboardURL}}" class="cta-button">View Dashboard</a>
                <a href="{{.ImageDetailsURL}}" class="cta-button">Image Details</a>
            </div>
        </div>
        <div class="footer">
            <p>Image Factory - Secure Container Platform for Modern Development</p>
        </div>
    </div>
</body>
</html>`,
	}
}

// getImageDeprecatedTemplate returns the image deprecated notification template
func getImageDeprecatedTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: "image_deprecated",
		Category:     CategoryNotifications,
		Subject:      "Image {{.ImageName}} has been deprecated",
		Description:  "Notification when an image is deprecated",
		TextTemplate: `Image Deprecated

Hello,

The image "{{.ImageName}}" (version {{.Version}}) has been marked as deprecated.

IMAGE INFORMATION
=================
• Image Name: {{.ImageName}}
• Version: {{.Version}}
• Removal Date: {{.RemovalDate}}

ACTION REQUIRED
===============
Please update to the latest version to ensure continued support and security updates.

VIEW ALTERNATIVES
==================
Dashboard: {{.DashboardURL}}
Image Details: {{.ImageDetailsURL}}

For questions, contact support at support@imgfactory.com

Best regards,
The Image Factory Team`,
		HTMLTemplate: `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Image Deprecated</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; background-color: #f8f9fa; }
        .container { max-width: 600px; margin: 0 auto; background-color: white; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%); color: white; padding: 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { padding: 30px; }
        .info-box { background-color: #fef3c7; border-left: 4px solid #f59e0b; padding: 15px; margin: 20px 0; border-radius: 4px; }
        .info-box h3 { margin: 0 0 10px 0; color: #92400e; }
        .info-box p { margin: 5px 0; color: #78350f; }
        .cta-button { display: inline-block; background: #f59e0b; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; margin: 10px 5px 10px 0; }
        .footer { background-color: #f3f4f6; padding: 20px; text-align: center; font-size: 12px; color: #6b7280; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⚠️ Image Deprecated</h1>
        </div>
        <div class="content">
            <p>Hello,</p>
            <p>The image <strong>{{.ImageName}}</strong> (version {{.Version}}) has been marked as deprecated.</p>
            
            <div class="info-box">
                <h3>Image Information</h3>
                <p><strong>Image Name:</strong> {{.ImageName}}</p>
                <p><strong>Version:</strong> {{.Version}}</p>
                <p><strong>Removal Date:</strong> {{.RemovalDate}}</p>
            </div>
            
            <h3>Action Required</h3>
            <p>Please update to the latest version to ensure continued support and security updates.</p>
            
            <p style="text-align: center; margin-top: 30px;">
                <a href="{{.DashboardURL}}" class="cta-button">View Dashboard</a>
                <a href="{{.ImageDetailsURL}}" class="cta-button">Image Details</a>
            </p>
            
            <p style="margin-top: 30px; color: #6b7280; font-size: 14px;">
                For questions, contact support at <a href="mailto:support@imgfactory.com">support@imgfactory.com</a>
            </p>
        </div>
        <div class="footer">
            <p>Image Factory - Secure Container Platform for Modern Development</p>
        </div>
    </div>
</body>
</html>`,
	}
}

// getImageSecurityAlertTemplate returns the image security alert notification template
func getImageSecurityAlertTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: "image_security_alert",
		Category:     CategoryNotifications,
		Subject:      "Security vulnerability found in {{.ImageName}}",
		Description:  "Notification when a security vulnerability is detected",
		TextTemplate: `Security Alert

Hello,

A security vulnerability has been detected in image "{{.ImageName}}" (version {{.Version}}).

VULNERABILITY DETAILS
=====================
• Image Name: {{.ImageName}}
• Version: {{.Version}}
• Severity: {{.Severity}}
• CVE ID: {{.CVEId}}
• Description: {{.VulnerabilityDescription}}

RECOMMENDED ACTION
==================
Please update to the latest version as soon as possible.

UPDATE NOW
==========
Dashboard: {{.DashboardURL}}
Image Details: {{.ImageDetailsURL}}

This vulnerability has been flagged as critical. Please prioritize this update.

For questions, contact security@imgfactory.com

Best regards,
The Image Factory Team`,
		HTMLTemplate: `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Security Alert</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; background-color: #f8f9fa; }
        .container { max-width: 600px; margin: 0 auto; background-color: white; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%); color: white; padding: 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 24px; }
        .alert-box { background-color: #fee2e2; border-left: 4px solid #ef4444; padding: 15px; margin: 20px 0; border-radius: 4px; }
        .alert-box h3 { margin: 0 0 10px 0; color: #7f1d1d; }
        .alert-box p { margin: 5px 0; color: #991b1b; }
        .severity { display: inline-block; padding: 4px 12px; border-radius: 20px; font-weight: bold; margin: 5px 0; }
        .severity-critical { background-color: #ef4444; color: white; }
        .cta-button { display: inline-block; background: #ef4444; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; margin: 10px 5px 10px 0; }
        .footer { background-color: #f3f4f6; padding: 20px; text-align: center; font-size: 12px; color: #6b7280; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔒 Security Alert</h1>
        </div>
        <div class="content" style="padding: 30px;">
            <p>Hello,</p>
            <p>A security vulnerability has been detected in image <strong>{{.ImageName}}</strong> (version {{.Version}}).</p>
            
            <div class="alert-box">
                <h3>Vulnerability Details</h3>
                <p><strong>Image Name:</strong> {{.ImageName}}</p>
                <p><strong>Version:</strong> {{.Version}}</p>
                <p><span class="severity severity-critical">{{.Severity}}</span></p>
                <p><strong>CVE ID:</strong> {{.CVEId}}</p>
                <p><strong>Description:</strong> {{.VulnerabilityDescription}}</p>
            </div>
            
            <h3 style="color: #991b1b;">Recommended Action</h3>
            <p><strong>Please update to the latest version as soon as possible.</strong></p>
            
            <p style="text-align: center; margin-top: 30px;">
                <a href="{{.DashboardURL}}" class="cta-button">View Dashboard</a>
                <a href="{{.ImageDetailsURL}}" class="cta-button">Image Details</a>
            </p>
            
            <p style="margin-top: 30px; padding: 15px; background-color: #fee2e2; border-radius: 4px; color: #7f1d1d;">
                <strong>Important:</strong> This vulnerability has been flagged as critical. Please prioritize this update.
            </p>
            
            <p style="margin-top: 20px; color: #6b7280; font-size: 14px;">
                For questions, contact <a href="mailto:security@imgfactory.com">security@imgfactory.com</a>
            </p>
        </div>
        <div class="footer">
            <p>Image Factory - Secure Container Platform for Modern Development</p>
        </div>
    </div>
</body>
</html>`,
	}
}

// getImagePublishedTemplate returns the image published notification template
func getImagePublishedTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: "image_published",
		Category:     CategoryNotifications,
		Subject:      "New image {{.ImageName}} published to the catalog",
		Description:  "Notification when a new image is published",
		TextTemplate: `New Image Published

Hello,

A new image has been published to the Image Factory catalog.

IMAGE DETAILS
=============
• Image Name: {{.ImageName}}
• Version: {{.Version}}
• Publisher: {{.Publisher}}
• Published At: {{.PublishedAt}}
• Description: {{.Description}}

GET STARTED
===========
You can now use this image in your projects. Pull it using:

docker pull {{.RegistryURL}}/{{.ImageName}}:{{.Version}}

LEARN MORE
==========
Dashboard: {{.DashboardURL}}
Image Details: {{.ImageDetailsURL}}
Documentation: {{.DocumentationURL}}

Start using this image in your builds today!

Best regards,
The Image Factory Team`,
		HTMLTemplate: `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Image Published</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; background-color: #f8f9fa; }
        .container { max-width: 600px; margin: 0 auto; background-color: white; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #3b82f6 0%, #1e40af 100%); color: white; padding: 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 24px; }
        .info-box { background-color: #dbeafe; border-left: 4px solid #3b82f6; padding: 15px; margin: 20px 0; border-radius: 4px; }
        .info-box h3 { margin: 0 0 10px 0; color: #1e40af; }
        .info-box p { margin: 5px 0; color: #1e3a8a; }
        .command-box { background-color: #1f2937; color: #10b981; padding: 15px; border-radius: 4px; font-family: monospace; margin: 15px 0; word-break: break-all; }
        .cta-button { display: inline-block; background: #3b82f6; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; margin: 10px 5px 10px 0; }
        .footer { background-color: #f3f4f6; padding: 20px; text-align: center; font-size: 12px; color: #6b7280; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎉 New Image Published</h1>
        </div>
        <div class="content" style="padding: 30px;">
            <p>Hello,</p>
            <p>A new image has been published to the Image Factory catalog.</p>
            
            <div class="info-box">
                <h3>Image Details</h3>
                <p><strong>Image Name:</strong> {{.ImageName}}</p>
                <p><strong>Version:</strong> {{.Version}}</p>
                <p><strong>Publisher:</strong> {{.Publisher}}</p>
                <p><strong>Published At:</strong> {{.PublishedAt}}</p>
                <p><strong>Description:</strong> {{.Description}}</p>
            </div>
            
            <h3>Get Started</h3>
            <p>You can now use this image in your projects. Pull it using:</p>
            <div class="command-box">docker pull {{.RegistryURL}}/{{.ImageName}}:{{.Version}}</div>
            
            <p style="text-align: center; margin-top: 30px;">
                <a href="{{.DashboardURL}}" class="cta-button">View Dashboard</a>
                <a href="{{.ImageDetailsURL}}" class="cta-button">Image Details</a>
                <a href="{{.DocumentationURL}}" class="cta-button">Documentation</a>
            </p>
            
            <p style="margin-top: 30px; text-align: center; color: #3b82f6; font-weight: bold;">
                Start using this image in your builds today!
            </p>
        </div>
        <div class="footer">
            <p>Image Factory - Secure Container Platform for Modern Development</p>
        </div>
    </div>
</body>
</html>`,
	}
}
