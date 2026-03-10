package main

// getUserManagementTemplates returns user management related templates
func getUserManagementTemplates() []EmailTemplate {
	return []EmailTemplate{
		getTenantOnboardingTemplate(),
		getUserAddedToTenantTemplate(),
		getUserRoleChangedTemplate(),
		getUserSuspendedTemplate(),
		getUserActivatedTemplate(),
	}
}

// getTenantOnboardingTemplate returns the tenant onboarding template
func getTenantOnboardingTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: TemplateTenantOnboarding,
		Category:     CategoryUserManagement,
		Subject:      "Welcome to Image Factory - Your Account is Ready!",
		Description:  "Professional tenant onboarding template for new tenants",
		TextTemplate: `Welcome to Image Factory!

Hello {{.ContactName}},

We are excited to inform you that your tenant account has been successfully created and is ready to use.

TENANT INFORMATION
==================
• Tenant Name: {{.TenantName}}
• Tenant ID: {{.TenantID}}
• Industry: {{.Industry}}
• Country: {{.Country}}

YOUR QUOTAS
===========
• API Rate Limit: {{.APIRateLimit}} requests/minute
• Storage Limit: {{.StorageLimit}} GB
• Users: Up to {{.MaxUsers}} users

GET STARTED
===========
Log in to your dashboard: {{.DashboardURL}}

Use your credentials to access the Image Factory platform and start building container images.

NEED HELP?
==========
If you have any questions, please contact our support team.
📧 Email: support@imagefactory.local
📚 Docs: https://docs.imagefactory.local
🔗 Status: https://status.imagefactory.local

Best regards,
The Image Factory Team

---
Image Factory - Secure Container Platform for Modern Development`,
		HTMLTemplate: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to Image Factory</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            margin: 0;
            padding: 0;
            background-color: #f8fafc;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background-color: #ffffff;
            border-radius: 12px;
            overflow: hidden;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            padding: 40px 30px;
            text-align: center;
            color: white;
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 600;
        }
        .header p {
            margin: 8px 0 0 0;
            opacity: 0.9;
            font-size: 16px;
        }
        .content {
            padding: 40px 30px;
        }
        .welcome-message {
            font-size: 18px;
            margin-bottom: 30px;
            color: #374151;
        }
        .info-section {
            background-color: #f9fafb;
            border-radius: 8px;
            padding: 24px;
            margin: 24px 0;
            border-left: 4px solid #667eea;
        }
        .info-section h3 {
            margin: 0 0 16px 0;
            color: #1f2937;
            font-size: 18px;
            font-weight: 600;
        }
        .info-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 16px;
        }
        .info-item {
            display: flex;
            flex-direction: column;
        }
        .info-label {
            font-weight: 600;
            color: #6b7280;
            font-size: 12px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 4px;
        }
        .info-value {
            color: #1f2937;
            font-size: 14px;
            font-weight: 500;
        }
        .quota-section {
            background-color: #ecfdf5;
            border-radius: 8px;
            padding: 24px;
            margin: 24px 0;
            border-left: 4px solid #10b981;
        }
        .quota-section h3 {
            margin: 0 0 16px 0;
            color: #065f46;
            font-size: 18px;
            font-weight: 600;
        }
        .quota-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 16px;
        }
        .quota-item {
            display: flex;
            align-items: center;
            padding: 12px;
            background-color: white;
            border-radius: 6px;
            border: 1px solid #d1fae5;
        }
        .quota-icon {
            width: 32px;
            height: 32px;
            background-color: #10b981;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            margin-right: 12px;
            color: white;
            font-weight: bold;
        }
        .quota-details h4 {
            margin: 0;
            font-size: 14px;
            font-weight: 600;
            color: #065f46;
        }
        .quota-details p {
            margin: 2px 0 0 0;
            font-size: 12px;
            color: #059669;
        }
        .cta-section {
            text-align: center;
            margin: 32px 0;
        }
        .cta-button {
            display: inline-block;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 16px 32px;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 600;
            font-size: 16px;
            box-shadow: 0 4px 14px rgba(102, 126, 234, 0.3);
            transition: transform 0.2s ease;
        }
        .cta-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(102, 126, 234, 0.4);
        }
        .support-section {
            background-color: #f3f4f6;
            padding: 24px;
            border-radius: 8px;
            margin: 24px 0;
            text-align: center;
        }
        .support-section h4 {
            margin: 0 0 8px 0;
            color: #374151;
            font-size: 16px;
            font-weight: 600;
        }
        .support-section p {
            margin: 0;
            color: #6b7280;
            font-size: 14px;
        }
        .footer {
            background-color: #1f2937;
            color: #9ca3af;
            padding: 32px 30px;
            text-align: center;
        }
        .footer h5 {
            margin: 0 0 8px 0;
            color: #f9fafb;
            font-size: 14px;
            font-weight: 600;
        }
        .footer p {
            margin: 0;
            font-size: 12px;
            line-height: 1.5;
        }
        .footer a {
            color: #667eea;
            text-decoration: none;
        }
        .footer a:hover {
            text-decoration: underline;
        }
        @media (max-width: 600px) {
            .container {
                margin: 10px;
                border-radius: 8px;
            }
            .header {
                padding: 30px 20px;
            }
            .content {
                padding: 30px 20px;
            }
            .info-grid {
                grid-template-columns: 1fr;
            }
            .quota-grid {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to Image Factory</h1>
            <p>Your account is ready to use</p>
        </div>

        <div class="content">
            <div class="welcome-message">
                <p>Hello <strong>{{.ContactName}}</strong>,</p>
                <p>We are excited to inform you that your tenant account has been successfully created and is ready to use.</p>
            </div>

            <div class="info-section">
                <h3>Tenant Information</h3>
                <div class="info-grid">
                    <div class="info-item">
                        <span class="info-label">Tenant Name</span>
                        <span class="info-value">{{.TenantName}}</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">Tenant ID</span>
                        <span class="info-value">{{.TenantID}}</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">Industry</span>
                        <span class="info-value">{{.Industry}}</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">Country</span>
                        <span class="info-value">{{.Country}}</span>
                    </div>
                </div>
            </div>

            <div class="quota-section">
                <h3>Your Quotas</h3>
                <div class="quota-grid">
                    <div class="quota-item">
                        <div class="quota-icon">⚡</div>
                        <div class="quota-details">
                            <h4>API Rate Limit</h4>
                            <p>{{.APIRateLimit}} req/min</p>
                        </div>
                    </div>
                    <div class="quota-item">
                        <div class="quota-icon">💾</div>
                        <div class="quota-details">
                            <h4>Storage Limit</h4>
                            <p>{{.StorageLimit}} GB</p>
                        </div>
                    </div>
                    <div class="quota-item">
                        <div class="quota-icon">👥</div>
                        <div class="quota-details">
                            <h4>Max Users</h4>
                            <p>Up to {{.MaxUsers}}</p>
                        </div>
                    </div>
                </div>
            </div>

            <div class="cta-section">
                <a href="{{.DashboardURL}}" class="cta-button">Access Your Dashboard</a>
                <p style="margin: 16px 0 0 0; color: #6b7280; font-size: 14px;">
                    Use your credentials to log in and start building
                </p>
            </div>

            <div class="support-section">
                <h4>Need Help Getting Started?</h4>
                <p>Our support team is here to help you get the most out of Image Factory.</p>
            </div>
        </div>

        <div class="footer">
            <h5>Image Factory</h5>
            <p>Secure Container Platform for Modern Development<br>
            <a href="mailto:support@imagefactory.local">support@imagefactory.local</a> | <a href="#">Documentation</a> | <a href="#">Status Page</a></p>
            <p style="margin-top: 16px; font-size: 11px;">
                You received this email because your tenant account was created in Image Factory.
            </p>
        </div>
    </div>
</body>
</html>`,
	}
}

// getUserAddedToTenantTemplate returns the user added to tenant template
func getUserAddedToTenantTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: TemplateUserAddedToTenant,
		Category:     CategoryUserManagement,
		Subject:      "Welcome to {{.TenantName}} on Image Factory",
		Description:  "Notification when a user is added to a tenant",
		TextTemplate: `Welcome to {{.TenantName}}

Hello {{.UserName}},

You have been added to {{.TenantName}} on Image Factory, the secure container platform for modern development.

TENANT ASSIGNMENT
=================
• Tenant: {{.TenantName}}
• Your Role: {{.Role}}

START HERE
==========
Access your dashboard: {{.DashboardURL}}

Your account is ready to use. Log in with your existing credentials to start collaborating with your team.

NEED HELP?
==========
If you have any questions or need assistance, contact your tenant administrator or our support team.

Best regards,
The Image Factory Team

---
Image Factory - Secure Container Platform for Modern Development
support@imagefactory.local | Documentation | Status Page`,
		HTMLTemplate: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to {{.TenantName}}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            margin: 0;
            padding: 0;
            background-color: #f8fafc;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background-color: #ffffff;
            border-radius: 12px;
            overflow: hidden;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        .header {
            background: linear-gradient(135deg, #10b981 0%, #059669 100%);
            padding: 40px 30px;
            text-align: center;
            color: white;
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 600;
        }
        .header p {
            margin: 8px 0 0 0;
            opacity: 0.9;
            font-size: 14px;
        }
        .content {
            padding: 40px 30px;
        }
        .welcome-message {
            font-size: 16px;
            margin-bottom: 24px;
            color: #374151;
        }
        .tenant-card {
            background-color: #f9fafb;
            border-radius: 8px;
            padding: 24px;
            margin: 24px 0;
            border-left: 4px solid #10b981;
        }
        .tenant-card h3 {
            margin: 0 0 16px 0;
            color: #1f2937;
            font-size: 18px;
            font-weight: 600;
        }
        .info-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 16px;
        }
        .info-item {
            display: flex;
            flex-direction: column;
        }
        .info-label {
            font-weight: 600;
            color: #6b7280;
            font-size: 12px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 4px;
        }
        .info-value {
            color: #1f2937;
            font-size: 14px;
            font-weight: 500;
        }
        .cta-section {
            text-align: center;
            margin: 32px 0;
        }
        .cta-button {
            display: inline-block;
            background: #10b981;
            color: white;
            padding: 12px 24px;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            font-size: 14px;
        }
        .help-section {
            background-color: #f0fdf4;
            border-radius: 8px;
            padding: 24px;
            margin: 24px 0;
            color: #374151;
        }
        .footer {
            background-color: #1f2937;
            color: #9ca3af;
            padding: 32px 30px;
            text-align: center;
            font-size: 12px;
        }
        .footer a {
            color: #10b981;
            text-decoration: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to {{.TenantName}}</h1>
            <p>You're ready to get started</p>
        </div>
        <div class="content">
            <p class="welcome-message">Hello {{.UserName}},</p>
            <p class="welcome-message">You have been added to <strong>{{.TenantName}}</strong> on Image Factory. Your account is now active and ready to use.</p>

            <div class="tenant-card">
                <h3>Your Tenant Details</h3>
                <div class="info-grid">
                    <div class="info-item">
                        <span class="info-label">Tenant</span>
                        <span class="info-value">{{.TenantName}}</span>
                    </div>
                    <div class="info-item">
                        <span class="info-label">Your Role</span>
                        <span class="info-value">{{.Role}}</span>
                    </div>
                </div>
            </div>

            <div class="cta-section">
                <a href="{{.DashboardURL}}" class="cta-button">Go to Dashboard</a>
            </div>

            <p>You can now log in with your existing credentials and start collaborating with your team. All your projects, builds, deployments, and images are waiting for you.</p>

            <div class="help-section">
                <strong>Need Help?</strong>
                <p style="margin: 8px 0 0 0;">If you have any questions or need assistance getting started, please don't hesitate to contact your tenant administrator or our support team.</p>
            </div>
        </div>
        <div class="footer">
            <p>Image Factory - Secure Container Platform for Modern Development</p>
            <p><a href="mailto:support@imagefactory.local">support@imagefactory.local</a></p>
        </div>
    </div>
</body>
</html>`,
	}
}

// getUserRoleChangedTemplate returns the user role changed notification template
func getUserRoleChangedTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: "user_role_changed",
		Category:     CategoryUserManagement,
		Subject:      "Your role in {{.TenantName}} has been updated",
		Description:  "Notification when a user's role in a tenant is changed",
		TextTemplate: `Your Role Has Changed

Hello {{.UserName}},

Your role in {{.TenantName}} has been changed.

ROLE UPDATE
===========
• Tenant: {{.TenantName}}
• Previous Role: {{.OldRole}}
• New Role: {{.NewRole}}

Your new role grants you updated permissions and access levels within {{.TenantName}}. Please review your new capabilities in the dashboard.

ACCESS YOUR ACCOUNT
===================
Visit the dashboard: {{.DashboardURL}}

If you have any questions about your role change or the new permissions, please contact your tenant administrator or our support team.

Best regards,
The Image Factory Team

---
Image Factory - Secure Container Platform for Modern Development`,
		HTMLTemplate: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Your role has been updated</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            margin: 0;
            padding: 0;
            background-color: #f8fafc;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background-color: #ffffff;
            border-radius: 12px;
            overflow: hidden;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        .header {
            background: linear-gradient(135deg, #f59e0b 0%, #f97316 100%);
            padding: 40px 30px;
            text-align: center;
            color: white;
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 600;
        }
        .header p {
            margin: 8px 0 0 0;
            opacity: 0.9;
            font-size: 16px;
        }
        .content {
            padding: 40px 30px;
        }
        .role-card {
            background-color: #fef3c7;
            border-left: 4px solid #f59e0b;
            border-radius: 8px;
            padding: 24px;
            margin: 24px 0;
        }
        .role-card h3 {
            margin: 0 0 16px 0;
            color: #92400e;
            font-size: 18px;
            font-weight: 600;
        }
        .role-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 16px;
        }
        .role-item {
            display: flex;
            flex-direction: column;
        }
        .role-label {
            font-weight: 600;
            color: #b45309;
            font-size: 12px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 4px;
        }
        .role-value {
            color: #7c2d12;
            font-size: 14px;
            font-weight: 500;
        }
        .role-change {
            background-color: #f3f4f6;
            padding: 12px;
            border-radius: 6px;
            margin: 8px 0;
            font-size: 13px;
        }
        .old-role {
            color: #6b7280;
            text-decoration: line-through;
        }
        .new-role {
            color: #059669;
            font-weight: 600;
        }
        .cta-section {
            text-align: center;
            margin: 32px 0;
        }
        .cta-button {
            display: inline-block;
            background: #f59e0b;
            color: white;
            padding: 12px 24px;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            font-size: 14px;
            margin: 8px;
        }
        .info-section {
            background-color: #f9fafb;
            border-radius: 8px;
            padding: 16px;
            margin: 16px 0;
            font-size: 14px;
            line-height: 1.5;
        }
        .footer {
            background-color: #1f2937;
            color: #9ca3af;
            padding: 32px 30px;
            text-align: center;
            font-size: 12px;
        }
        .footer a {
            color: #fbbf24;
            text-decoration: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Your Role Updated</h1>
            <p>{{.TenantName}}</p>
        </div>
        <div class="content">
            <p>Hello {{.UserName}},</p>
            <p>Your role in <strong>{{.TenantName}}</strong> has been changed. Your permissions and access levels have been updated accordingly.</p>

            <div class="role-card">
                <h3>Role Change Details</h3>
                <div class="role-grid">
                    <div class="role-item">
                        <span class="role-label">Tenant</span>
                        <span class="role-value">{{.TenantName}}</span>
                    </div>
                    <div class="role-item">
                        <span class="role-label">Updated On</span>
                        <span class="role-value">Today</span>
                    </div>
                </div>
                <div class="role-change" style="margin-top: 16px;">
                    <span class="old-role">{{.OldRole}}</span> → <span class="new-role">{{.NewRole}}</span>
                </div>
            </div>

            <div class="info-section">
                <strong>What's next?</strong>
                <p style="margin: 8px 0 0 0;">Visit the dashboard to review your updated permissions, access new features, and collaborate with your team. Your credentials remain the same.</p>
            </div>

            <div class="cta-section">
                <a href="{{.DashboardURL}}" class="cta-button">Visit Dashboard</a>
            </div>

            <p>If you have any questions about your role change or need assistance, please contact your tenant administrator or reach out to our support team.</p>
        </div>
        <div class="footer">
            <p>Image Factory - Secure Container Platform for Modern Development</p>
            <p><a href="mailto:support@imagefactory.local">support@imagefactory.local</a></p>
        </div>
    </div>
</body>
</html>`,
	}
}

// getUserSuspendedTemplate returns the user suspended notification template
func getUserSuspendedTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: TemplateUserSuspended,
		Category:     CategoryUserManagement,
		Subject:      "Your Image Factory Account Has Been Suspended",
		Description:  "Notification sent when a user account is suspended",
		TextTemplate: `Account Suspended

Hello {{.UserName}},

Your Image Factory account has been suspended.

ACCOUNT INFORMATION
===================
• Email: {{.UserEmail}}
• Tenant: {{.TenantName}}
• Suspended At: {{.SuspendedAt}}

REASON FOR SUSPENSION
=====================
{{.Reason}}

WHAT THIS MEANS
===============
• You will not be able to log in to the platform
• Your access to all projects and resources is temporarily disabled
• Your data and settings are preserved

TO REACTIVATE YOUR ACCOUNT
=========================
Please contact your tenant administrator or support team to resolve this issue.

NEED HELP?
==========
If you believe this suspension was made in error, please contact:
📧 Email: support@imagefactory.local
📚 Docs: https://docs.imagefactory.local

Best regards,
The Image Factory Team

---
Image Factory - Secure Container Platform for Modern Development`,
		HTMLTemplate: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Account Suspended</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            margin: 0;
            padding: 0;
            background-color: #f8fafc;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background-color: #ffffff;
            border-radius: 12px;
            overflow: hidden;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        .header {
            background: linear-gradient(135deg, #dc3545 0%, #c82333 100%);
            padding: 40px 30px;
            text-align: center;
            color: white;
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 600;
        }
        .header p {
            margin: 8px 0 0 0;
            opacity: 0.9;
            font-size: 16px;
        }
        .content {
            padding: 40px 30px;
        }
        .account-info {
            background-color: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 8px;
            padding: 20px;
            margin: 20px 0;
        }
        .account-info h3 {
            margin-top: 0;
            color: #495057;
            font-size: 16px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .account-info p {
            margin: 8px 0;
            font-size: 14px;
        }
        .reason-box {
            background-color: #fff3cd;
            border: 1px solid #ffeaa7;
            border-radius: 8px;
            padding: 20px;
            margin: 20px 0;
        }
        .reason-box h3 {
            margin-top: 0;
            color: #856404;
        }
        .warning-box {
            background-color: #f8d7da;
            border: 1px solid #f5c6cb;
            border-radius: 8px;
            padding: 20px;
            margin: 20px 0;
        }
        .warning-box h3 {
            margin-top: 0;
            color: #721c24;
        }
        .help-section {
            background-color: #e7f3ff;
            border: 1px solid #b3d7ff;
            border-radius: 8px;
            padding: 20px;
            margin: 20px 0;
        }
        .help-section h3 {
            margin-top: 0;
            color: #004085;
        }
        .footer {
            background-color: #f8f9fa;
            padding: 30px;
            text-align: center;
            border-top: 1px solid #e9ecef;
        }
        .footer p {
            margin: 5px 0;
            color: #6c757d;
            font-size: 14px;
        }
        .footer a {
            color: #007bff;
            text-decoration: none;
        }
        .footer a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Account Suspended</h1>
            <p>Your access has been temporarily disabled</p>
        </div>
        <div class="content">
            <p>Hello {{.UserName}},</p>

            <p>Your Image Factory account has been suspended. This means you currently cannot access the platform.</p>

            <div class="account-info">
                <h3>Account Information</h3>
                <p><strong>Email:</strong> {{.UserEmail}}</p>
                <p><strong>Tenant:</strong> {{.TenantName}}</p>
                <p><strong>Suspended At:</strong> {{.SuspendedAt}}</p>
            </div>

            <div class="reason-box">
                <h3>Reason for Suspension</h3>
                <p>{{.Reason}}</p>
            </div>

            <div class="warning-box">
                <h3>What This Means</h3>
                <ul>
                    <li>You will not be able to log in to the platform</li>
                    <li>Your access to all projects and resources is temporarily disabled</li>
                    <li>Your data and settings are preserved and will be restored when your account is reactivated</li>
                </ul>
            </div>

            <div class="help-section">
                <h3>To Reactivate Your Account</h3>
                <p>Please contact your tenant administrator or our support team to resolve this issue.</p>
            </div>

            <div class="help-section">
                <h3>Need Help?</h3>
                <p>If you believe this suspension was made in error, please contact our support team:</p>
                <p>📧 <a href="mailto:support@imagefactory.local">support@imagefactory.local</a><br>
                📚 <a href="https://docs.imagefactory.local">Documentation</a></p>
            </div>
        </div>
        <div class="footer">
            <p><strong>Image Factory</strong> - Secure Container Platform for Modern Development</p>
            <p><a href="mailto:support@imagefactory.local">support@imagefactory.local</a></p>
        </div>
    </div>
</body>
</html>`,
	}
}

// getUserActivatedTemplate returns the user activated notification template
func getUserActivatedTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: TemplateUserActivated,
		Category:     CategoryUserManagement,
		Subject:      "Your Image Factory Account Has Been Reactivated",
		Description:  "Notification sent when a suspended user account is reactivated",
		TextTemplate: `Account Reactivated

Hello {{.UserName}},

Your Image Factory account has been reactivated and you can now access the platform again.

ACCOUNT INFORMATION
===================
• Email: {{.UserEmail}}
• Tenant: {{.TenantName}}
• Reactivated At: {{.ReactivatedAt}}

WHAT THIS MEANS
===============
• You can now log in to the platform
• Your access to all projects and resources has been restored
• All your previous data and settings are available

GET STARTED
===========
Log in to your dashboard: {{.DashboardURL}}

If you have any questions about your account reactivation, please contact your tenant administrator.

NEED HELP?
==========
📧 Email: support@imagefactory.local
📚 Docs: https://docs.imagefactory.local

Welcome back!
The Image Factory Team

---
Image Factory - Secure Container Platform for Modern Development`,
		HTMLTemplate: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Account Reactivated</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            margin: 0;
            padding: 0;
            background-color: #f8fafc;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background-color: #ffffff;
            border-radius: 12px;
            overflow: hidden;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        .header {
            background: linear-gradient(135deg, #28a745 0%, #20c997 100%);
            padding: 40px 30px;
            text-align: center;
            color: white;
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 600;
        }
        .header p {
            margin: 8px 0 0 0;
            opacity: 0.9;
            font-size: 16px;
        }
        .content {
            padding: 40px 30px;
        }
        .account-info {
            background-color: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 8px;
            padding: 20px;
            margin: 20px 0;
        }
        .account-info h3 {
            margin-top: 0;
            color: #495057;
            font-size: 16px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .account-info p {
            margin: 8px 0;
            font-size: 14px;
        }
        .success-box {
            background-color: #d4edda;
            border: 1px solid #c3e6cb;
            border-radius: 8px;
            padding: 20px;
            margin: 20px 0;
        }
        .success-box h3 {
            margin-top: 0;
            color: #155724;
        }
        .get-started {
            background-color: #e7f3ff;
            border: 1px solid #b3d7ff;
            border-radius: 8px;
            padding: 20px;
            margin: 20px 0;
            text-align: center;
        }
        .get-started h3 {
            margin-top: 0;
            color: #004085;
        }
        .get-started a {
            display: inline-block;
            background-color: #007bff;
            color: white;
            padding: 12px 24px;
            text-decoration: none;
            border-radius: 6px;
            margin: 10px 0;
            font-weight: 500;
        }
        .get-started a:hover {
            background-color: #0056b3;
        }
        .help-section {
            background-color: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 8px;
            padding: 20px;
            margin: 20px 0;
        }
        .help-section h3 {
            margin-top: 0;
            color: #495057;
        }
        .footer {
            background-color: #f8f9fa;
            padding: 30px;
            text-align: center;
            border-top: 1px solid #e9ecef;
        }
        .footer p {
            margin: 5px 0;
            color: #6c757d;
            font-size: 14px;
        }
        .footer a {
            color: #007bff;
            text-decoration: none;
        }
        .footer a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Account Reactivated</h1>
            <p>Welcome back to Image Factory!</p>
        </div>
        <div class="content">
            <p>Hello {{.UserName}},</p>

            <p>Great news! Your Image Factory account has been reactivated and you can now access the platform again.</p>

            <div class="account-info">
                <h3>Account Information</h3>
                <p><strong>Email:</strong> {{.UserEmail}}</p>
                <p><strong>Tenant:</strong> {{.TenantName}}</p>
                <p><strong>Reactivated At:</strong> {{.ReactivatedAt}}</p>
            </div>

            <div class="success-box">
                <h3>What This Means</h3>
                <ul>
                    <li>You can now log in to the platform</li>
                    <li>Your access to all projects and resources has been restored</li>
                    <li>All your previous data and settings are available</li>
                </ul>
            </div>

            <div class="get-started">
                <h3>Get Started</h3>
                <p>Log in to your dashboard to continue where you left off:</p>
                <a href="{{.DashboardURL}}">Access Dashboard</a>
            </div>

            <div class="help-section">
                <h3>Need Help?</h3>
                <p>If you have any questions about your account reactivation, please contact your tenant administrator or our support team:</p>
                <p>📧 <a href="mailto:support@imagefactory.local">support@imagefactory.local</a><br>
                📚 <a href="https://docs.imagefactory.local">Documentation</a></p>
            </div>
        </div>
        <div class="footer">
            <p><strong>Image Factory</strong> - Secure Container Platform for Modern Development</p>
            <p><a href="mailto:support@imagefactory.local">support@imagefactory.local</a></p>
        </div>
    </div>
</body>
</html>`,
	}
}
