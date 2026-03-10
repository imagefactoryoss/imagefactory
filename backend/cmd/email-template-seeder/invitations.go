package main

// getInvitationTemplates returns invitation related templates
func getInvitationTemplates() []EmailTemplate {
	return []EmailTemplate{
		getUserInvitationTemplate(),
	}
}

// getUserInvitationTemplate returns the user invitation template
func getUserInvitationTemplate() EmailTemplate {
	return EmailTemplate{
		TemplateType: TemplateUserInvitation,
		Category:     CategoryInvitations,
		Subject:      "You're invited to join {{.TenantName}} on Image Factory",
		Description:  "Professional invitation template for new users joining a tenant",
		TextTemplate: `You're invited to join {{.TenantName}} on Image Factory

Hello,

You've been invited to join {{.TenantName}} on Image Factory, the secure container platform for modern development.

{{if .Message}}
PERSONAL MESSAGE FROM {{.InviterName}}
==========================================
{{.Message}}

{{end}}
ACCEPT YOUR INVITATION
======================
Click here to join: {{.InvitationURL}}

Create your account and start collaborating with your team.

IMPORTANT
=========
• This invitation expires on {{.ExpiresAt}}
• The link is unique to you and can only be used once
• Keep this invitation secure

SECURITY NOTICE
===============
This invitation link is unique to you and can only be used once. Keep it secure and don't share it with others.

Didn't expect this invitation? You can safely ignore this email.

Best regards,
The Image Factory Team

---
Image Factory - Secure Container Platform for Modern Development
support@example.com | Documentation | Status Page`,
		HTMLTemplate: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>You're invited to join {{.TenantName}}</title>
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
        .invitation-message {
            font-size: 16px;
            margin-bottom: 24px;
            color: #374151;
        }
        .tenant-card {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            border-radius: 12px;
            padding: 24px;
            margin: 24px 0;
            text-align: center;
            color: white;
        }
        .tenant-card h2 {
            margin: 0 0 8px 0;
            font-size: 24px;
            font-weight: 600;
        }
        .tenant-card p {
            margin: 0;
            opacity: 0.9;
            font-size: 14px;
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
        }
        .expiry-notice {
            background-color: #fef3c7;
            border: 1px solid #f59e0b;
            border-radius: 8px;
            padding: 16px;
            margin: 24px 0;
            text-align: center;
        }
        .expiry-notice strong {
            color: #92400e;
        }
        .link-section {
            background-color: #f3f4f6;
            border-radius: 8px;
            padding: 20px;
            margin: 24px 0;
            border: 1px solid #e5e7eb;
        }
        .link-url {
            word-break: break-all;
            background-color: white;
            padding: 12px;
            border-radius: 6px;
            border: 1px solid #d1d5db;
            font-family: monospace;
            font-size: 12px;
            color: #374151;
        }
        .security-notice {
            background-color: #ecfdf5;
            border: 1px solid #10b981;
            border-radius: 8px;
            padding: 16px;
            margin: 24px 0;
        }
        .security-notice h4 {
            margin: 0 0 8px 0;
            color: #065f46;
            font-size: 14px;
            font-weight: 600;
        }
        .security-notice p {
            margin: 0;
            color: #047857;
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
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>You're Invited!</h1>
            <p>Join {{.TenantName}} on Image Factory</p>
        </div>

        <div class="content">
            <div class="invitation-message">
                <p>Hello,</p>
                <p>You've been invited to join a team on Image Factory, the secure container platform for modern development.</p>
            </div>

            <div class="tenant-card">
                <h2>{{.TenantName}}</h2>
                <p>Container Platform Team</p>
            </div>

            <div class="cta-section">
                <a href="{{.InvitationURL}}" class="cta-button">Accept Invitation</a>
                <p style="margin: 16px 0 0 0; color: #6b7280; font-size: 14px;">
                    Create your account and start collaborating
                </p>
            </div>

            <div class="expiry-notice">
                <strong>⏰ This invitation expires on {{.ExpiresAt}}</strong><br>
                <span style="font-size: 14px; color: #92400e;">Make sure to accept it before the expiration date</span>
            </div>

            <div class="link-section">
                <p>If the button doesn't work, copy and paste this link into your browser:</p>
                <div class="link-url">{{.InvitationURL}}</div>
            </div>

            <div class="security-notice">
                <h4>🔒 Secure Invitation</h4>
                <p>This invitation link is unique to you and can only be used once. Keep it secure and don't share it with others.</p>
            </div>
        </div>

        <div class="footer">
            <h5>Image Factory</h5>
            <p>Secure Container Platform for Modern Development<br>
            <a href="mailto:support@example.com">support@example.com</a> | <a href="#">Documentation</a> | <a href="#">Status Page</a></p>
            <p style="margin-top: 16px; font-size: 11px;">
                Didn't expect this invitation? You can safely ignore this email.
            </p>
        </div>
    </div>
</body>
</html>`,
	}
}
